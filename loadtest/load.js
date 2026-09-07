import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN;

const archive = open('./fixtures/test.zip', 'b');

const timeToResult = new Trend('time_to_result', true);
const jobFailed = new Rate('job_failed');

export const options = {
    scenarios: {
        jobs: {
            executor: 'ramping-arrival-rate',

            startRate: 2,
            timeUnit: '1s',

            preAllocatedVUs: 20,
            maxVUs: 150,

            stages: [
                { target: 2, duration: '30s' },
                { target: 5, duration: '30s' },
                { target: 10, duration: '30s' },
                { target: 20, duration: '30s' },
                { target: 20, duration: '30s' },
            ],

            gracefulStop: '30s',
        },
    },

    thresholds: {
        job_failed: ['rate<0.01'],
        time_to_result: ['p(95)<5000'],
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