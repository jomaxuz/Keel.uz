// The load test, kept in the repository so the next one is the same test.
//
// ⚠️ **This is why it exists.** The run of 2026-09-03 measured ~350 req/s and
// named mongod as the bottleneck, and the fixes that followed are meant to be
// judged against that number — but the script that produced it was written in a
// terminal and lost with the session. A second measurement taken with a
// different script is not a comparison, it is two unrelated numbers, and the
// difference between "the cache tripled throughput" and "I wrote an easier
// test" is invisible from the outside.
//
// Run it with k6:
//
//   k6 run -e BASE=https://b5somsa.keel.uz scripts/loadtest.js
//   k6 run -e BASE=http://127.0.0.1:8080 -e STAGE=smoke scripts/loadtest.js
//
// ⚠️ **From a second machine when the number has to be right.** The 2026-09-03
// run had the generator on the server under test, where it took one or two of
// the four cores; every figure from it is therefore conservative, and a rerun
// from the same place is only comparable to that one — not to the truth.

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const BASE = __ENV.BASE || 'http://127.0.0.1:8080';
const API = `${BASE}/api/v1`;

// ⚠️ **Read-only by default, and this is not caution for its own sake.** The
// obvious target is a live tenant, and the write path this would exercise is
// order creation: real receipt numbers, a kitchen screen filling with food
// nobody ordered, and a night's reports that cannot be told apart from a night's
// trade. Set `-e WRITE=1` only against a tenant that is yours to fill with
// rubbish (b5somsa is the test tenant — see the memory note).
const WRITE = __ENV.WRITE === '1';

// How much of the load is being served without touching Mongo. **The point of
// the whole exercise**: the throughput number alone cannot say whether the cache
// is working or whether the box simply had a quiet moment.
const cacheHits = new Rate('cache_hit_rate');
const cacheable = new Counter('cacheable_requests');

const STAGES = {
  // Ramp until something gives, the way the first run found the ceiling.
  ramp: [
    { duration: '1m', target: 50 },
    { duration: '2m', target: 200 },
    { duration: '2m', target: 500 },
    { duration: '1m', target: 0 },
  ],
  // The number that goes in the report: what it sustains without errors.
  shift: [
    { duration: '30s', target: 100 },
    { duration: '3m', target: 100 },
    { duration: '30s', target: 0 },
  ],
  // Thirty seconds to prove the script and the target work at all, before
  // anybody spends ten minutes measuring nothing.
  smoke: [{ duration: '30s', target: 5 }],
};

export const options = {
  stages: STAGES[__ENV.STAGE || 'shift'],
  thresholds: {
    // Deliberately loose. This is a measurement, not a gate: a run that stops
    // early because it crossed a line tells us less than one that finishes and
    // shows where it went.
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<5000'],
  },
};

// One visitor's flow, in the order a browser actually asks. Five of these nine
// requests are the cacheable public reads — the ones this whole change is
// about.
function visit() {
  const reads = [
    ['restaurant', `${API}/restaurant`],
    ['categories', `${API}/categories`],
    ['menu', `${API}/menu`],
    ['promotions', `${API}/promotions`],
    ['payment-methods', `${API}/payment-methods`],
  ];

  group('public reads', () => {
    for (const [name, url] of reads) {
      const res = http.get(url, {
        headers: { 'Accept-Encoding': 'gzip, br' },
        tags: { name },
      });
      check(res, { [`${name} 200`]: (r) => r.status === 200 });
      // ⚠️ Read from the header rather than inferred from the timing: a fast
      // response is not proof of a cache hit, and on a quiet box every response
      // is fast.
      const mark = res.headers['X-Cache'] || res.headers['X-cache'];
      if (mark) {
        cacheable.add(1);
        cacheHits.add(mark === 'HIT');
      }
    }
  });

  // The two the guest reaches by doing something, and neither is cacheable:
  // a dish page, and the delivery price for an address.
  group('interactions', () => {
    const quote = http.post(
      `${API}/delivery/quote`,
      JSON.stringify({ lat: 41.311081, lng: 69.240562 }),
      { headers: { 'Content-Type': 'application/json' }, tags: { name: 'delivery/quote' } },
    );
    // Not checked for 200: a tenant with delivery switched off answers this
    // with an error, and that is a correct answer rather than a failed test.
    check(quote, { 'quote answered': (r) => r.status > 0 });
  });

  if (WRITE) {
    group('order', () => {
      const res = http.post(
        `${API}/orders`,
        JSON.stringify({
          type: 'pickup',
          customer: { name: 'k6', phone: '+998900000000' },
          items: [],
        }),
        { headers: { 'Content-Type': 'application/json' }, tags: { name: 'orders' } },
      );
      check(res, { 'order answered': (r) => r.status > 0 });
    });
  }

  // A person reads the menu before they do anything else. Without this the
  // test measures how fast k6 can loop, not how many guests the box holds.
  sleep(1 + Math.random() * 2);
}

export default function () {
  visit();
}

// ⚠️ **k6 replaces its own summary the moment this function exists**, so the
// four numbers the report is actually written from have to be printed here or
// they are lost — and the run that loses them has to be done again.
export function handleSummary(data) {
  const m = data.metrics;
  const num = (metric, field, digits = 1) =>
    m[metric] && m[metric].values[field] !== undefined
      ? m[metric].values[field].toFixed(digits)
      : '—';

  const rate = m.cache_hit_rate;
  const cache = rate
    ? `${(rate.values.rate * 100).toFixed(1)}%  (${rate.values.passes + rate.values.fails} cacheable requests)`
    : 'no X-Cache header seen — the cache is not in front of these routes';

  const lines = [
    '',
    `  throughput      ${num('http_reqs', 'rate')} req/s`,
    `  p95             ${num('http_req_duration', 'p(95)', 0)} ms`,
    `  errors          ${(Number(num('http_req_failed', 'rate', 4)) * 100).toFixed(2)}%`,
    `  cache hit rate  ${cache}`,
    '',
    '  Compare against docs/LOAD_TEST_2026-09-03.md — and only if this run was',
    '  driven from the same place as that one.',
    '',
  ];
  return { stdout: lines.join('\n') + '\n' };
}
