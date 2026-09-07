import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN;

const RATE = Number(__ENV.RATE || 2);
const DURATION = __ENV.DURATION || '60s';

const archive = open('./fixtures/test.zip', 'b');

const timeToResult = new Trend('time_to_result', true);
const jobFailed = new Rate('job_failed');

export const options = {
    scenarios: {
        constant_load: {
            executor: 'constant-arrival-rate',

            rate: RATE,
            timeUnit: '1s',
            duration: DURATION,

            preAllocatedVUs: 50,
            maxVUs: 150,

            gracefulStop: '2m',
        },
    },

    thresholds: {
        job_failed: ['rate<0.01'],
    },
};

export default function () {
    const startedAt = Date.now();

    const createRes = http.post(
        `${BASE_URL}/bulk_generate`,
        archive,
        {
            headers: {
                Authorization: `Bearer ${TOKEN}`,
                'Content-Type': 'application/zip',
            },
            tags: {
                endpoint: 'bulk_generate',
            },
            timeout: '30s',
        },
    );

    const created = check(createRes, {
        'job created': (r) => r.status === 200,
    });

    if (!created) {
        jobFailed.add(true);
        return;
    }

    let jobID;

    try {
        jobID = createRes.json('job_id');
    } catch (_) {
        jobFailed.add(true);
        return;
    }

    if (!jobID) {
        jobFailed.add(true);
        return;
    }

    while (true) {
        const statusRes = http.get(
            `${BASE_URL}/get_job_status?id=${jobID}`,
            {
                headers: {
                    Authorization: `Bearer ${TOKEN}`,
                },
                tags: {
                    endpoint: 'job_status',
                },
                timeout: '10s',
            },
        );

        if (statusRes.status !== 200) {
            jobFailed.add(true);
            return;
        }

        let status;

        try {
            status = statusRes.json('status');
        } catch (_) {
            jobFailed.add(true);
            return;
        }

        if (status === 'completed') {
            timeToResult.add(Date.now() - startedAt);
            jobFailed.add(false);
            return;
        }

        if (status === 'failed') {
            timeToResult.add(Date.now() - startedAt);
            jobFailed.add(true);
            return;
        }

        sleep(0.1);
    }
}