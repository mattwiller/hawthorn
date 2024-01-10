# Hawthorn - Self-contained seed terminology service

Hawthorn contains everything required to set up a tiny, self-contained FHIR® terminology service. Currently, the
following operations are supported:

- [`GET /R4/CodeSystem/$lookup`](http://hl7.org/fhir/R4/codesystem-operation-lookup.html)

## Setup

To get started, [sign up for a UMLS Metathesaurus License](https://uts.nlm.nih.gov/uts/signup-login). This is required
for access to the underlying UMLS terminology data, and typically takes 1-3 days to process.

After receiving access to UMLS Metathesaurus, [your UMLS profile](https://uts.nlm.nih.gov/uts/edit-profile) contains the
API key needed for setting up the service.

Clone this repository, then build the service as a Docker container:

> **NOTE:** This will download the entire ~4 GB UMLS metathesaurus release, and run a build script to extract the
> code system data from the compressed file.

```bash
docker build --build-arg UMLS_API_KEY=$UMLS_API_KEY -t hawthorn:latest .
docker run -it -p '29927:29927' hawthorn:latest

# Alternatively, build the service directly
# This produces two primary output files:
# - umls.db, a sqlite DB containing code system data
# - hawthorn, the statically compiled server binary
make build
```

## Benchmark

Due to the "embedded" sqlite database, performance is excellent even at high load. To benchmark, `CodeSystem/$lookup`
repeated queries for a thousand random codes are sent as fast as possible over a single connection, to simulate usage of
the service as a sidecar. The benchmark was performed using [K6](https://k6.io/), see the [benchmark script](./k6.js)
for more details.

<details>
<summary><strong>tl;dr</strong> Served over 6,000 code lookups per second, with p99 latency under 0.5 ms and p99.99 latency under 5 ms</summary>

```
> k6 run --duration 7m --summary-trend-stats 'avg,min,med,p(75),p(90),p(95),p(99),p(99.9),p(99.99),max' k6.js

          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: k6.js

  scenarios: (100.00%) 1 scenario, 1 max VUs, 7m30s max duration (incl. graceful stop):
           * default: 1 looping VUs for 7m0s (gracefulStop: 30s)


     ✓ status was 200
     ✓ body was right

     checks.........................: 100.00% ✓ 5725472     ✗ 0
     data_received..................: 2.9 GB  7.0 MB/s
     data_sent......................: 425 MB  1.0 MB/s
     http_req_blocked...............: avg=684ns    min=299ns   med=559ns    p(75)=823ns    p(90)=1.08µs   p(95)=1.24µs
                                      p(99)=1.63µs   p(99.9)=6.57µs  p(99.99)=18.77µs max=3.27ms
     http_req_connecting............: avg=0ns      min=0s      med=0s       p(75)=0s       p(90)=0s       p(95)=0s
                                      p(99)=0s       p(99.9)=0s      p(99.99)=0s      max=159.48µs
     http_req_duration..............: avg=113.8µs  min=59.21µs med=100.94µs p(75)=120.48µs p(90)=149.22µs p(95)=178.66µs
                                      p(99)=313.97µs p(99.9)=1.34ms  p(99.99)=3.34ms  max=11.66ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 2862736
     http_req_receiving.............: avg=10.27µs  min=4.25µs  med=8.78µs   p(75)=12.08µs  p(90)=15.89µs  p(95)=18.82µs
                                      p(99)=27.24µs  p(99.9)=42.35µs p(99.99)=69.13µs max=5.26ms
     http_req_sending...............: avg=2.98µs   min=1.61µs  med=2.49µs   p(75)=3.47µs   p(90)=4.54µs   p(95)=5.34µs
                                      p(99)=7.05µs   p(99.9)=14.07µs p(99.99)=25.94µs max=912.98µs
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s       p(75)=0s       p(90)=0s       p(95)=0s
                                      p(99)=0s       p(99.9)=0s      p(99.99)=0s      max=0s
     http_req_waiting...............: avg=100.54µs min=51.32µs med=88.12µs  p(75)=106.22µs p(90)=133.02µs p(95)=161.41µs
                                      p(99)=287.89µs p(99.9)=1.31ms  p(99.99)=3.27ms  max=11.63ms
     http_reqs......................: 2862736 6816.037544/s
     iteration_duration.............: avg=143.9µs  min=79.09µs med=130.53µs p(75)=153.91µs p(90)=186.1µs  p(95)=216.42µs
                                      p(99)=354.75µs p(99.9)=1.4ms   p(99.99)=3.5ms   max=11.71ms
```

</details>

# License

Copyright 2024 Matthew Willer. Available for redistribution and use under the [BSD 3-Clause License](./LICENSE.txt)

FHIR® is the registered trademark of Health Level Seven International and the use does not constitute endorsement by HL7
