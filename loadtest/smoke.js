import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN;

const archive = open('./fixtures/test.zip', 'b');

const timeToResult = new Trend('time_to_result', true);
const jobFailed = new Rate('job_failed');

export const options = {
    vus: 1,
    iterations: 3,
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
        },
    );

    const created = check(createRes, {
        'job created': (r) => r.status === 200,
    });

    if (!created) {
        jobFailed.add(true);
        return;
    }

    const jobID = createRes.json('job_id');

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
            },
        );

        if (statusRes.status !== 200) {
            jobFailed.add(true);
            return;
        }

        const status = statusRes.json('status');

        if (status === 'completed') {
            timeToResult.add(Date.now() - startedAt);
            jobFailed.add(false);
            break;
        }

        if (status === 'failed') {
            timeToResult.add(Date.now() - startedAt);
            jobFailed.add(true);
            break;
        }

        sleep(0.1);
    }
}