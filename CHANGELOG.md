# Changelog

## [0.27.0](https://github.com/Simplifi-ED/grafana-apm-app/compare/clickhouse-apm-app-v0.26.1...clickhouse-apm-app-v0.27.0) (2026-10-28)


### Features

* **alerts:** post-parity B+C — Alerts UX + volume regression template ([#6](https://github.com/Simplifi-ED/grafana-apm-app/issues/6)) ([4e42836](https://github.com/Simplifi-ED/grafana-apm-app/commit/4e42836bc8f2edffbaab001946b8b56df14ddd69))
* **custom-metrics:** clickstack overview auto-charts via ch sql ([#9](https://github.com/Simplifi-ED/grafana-apm-app/issues/9)) ([c263035](https://github.com/Simplifi-ED/grafana-apm-app/commit/c2630359018fd74adc8adc58e11ed26f2a1e15dd))
* **FrameworkBadge:** add tests and refactor badge implementation ([07ea03a](https://github.com/Simplifi-ED/grafana-apm-app/commit/07ea03ad3204d60a5c760a331148713833050ab3))
* **issues:** server notify flags, Returned badge, Overview attention strip ([#7](https://github.com/Simplifi-ED/grafana-apm-app/issues/7)) ([5192f5c](https://github.com/Simplifi-ED/grafana-apm-app/commit/5192f5cbe3905ff57c7081a2635ee8a6bf23df37))
* **scorecard:** post-parity F — clickstack v2 readiness checks ([#10](https://github.com/Simplifi-ED/grafana-apm-app/issues/10)) ([8131673](https://github.com/Simplifi-ED/grafana-apm-app/commit/8131673827aae72bfeccd824b892331dc861f2b4))
* **skeleton:** add CardGridSkeleton and PanelSkeleton components for loading states ([ddad3c4](https://github.com/Simplifi-ED/grafana-apm-app/commit/ddad3c4d816fc840c0bb74529fe5b3d52ad5a0e2))
* **skeleton:** add SkeletonBar and TableSkeleton components for loading states ([aec8281](https://github.com/Simplifi-ED/grafana-apm-app/commit/aec828156c067f5416a428088b7816a59fb73006))


### Bug Fixes

* page triage annotations correctly + run CI for release-please PRs (spine A) ([#5](https://github.com/Simplifi-ED/grafana-apm-app/issues/5)) ([25b6415](https://github.com/Simplifi-ED/grafana-apm-app/commit/25b64150d95b5698bbec60c3e762d27524c0aea0))


### Documentation

* sync feature-matrix to post-parity spine A→F ([#11](https://github.com/Simplifi-ED/grafana-apm-app/issues/11)) ([25d9481](https://github.com/Simplifi-ED/grafana-apm-app/commit/25d94812f7792b7cdd59e9f096695651e6d663ad))

## [0.26.1](https://github.com/Simplifi-ED/grafana-apm-app/compare/clickhouse-apm-app-v0.26.0...clickhouse-apm-app-v0.26.1) (2026-10-28)


### Bug Fixes

* improve type handling in themeTypography tests ([1adc6f6](https://github.com/Simplifi-ED/grafana-apm-app/commit/1adc6f6cb4a929406d91e7df5dff75c30046ec85))

## [0.26.0](https://github.com/Simplifi-ED/grafana-apm-app/compare/clickhouse-apm-app-v0.25.3...clickhouse-apm-app-v0.26.0) (2026-10-28)


### Features

* add 'Has errors' and SDK/language filters to service inventory ([7e4a814](https://github.com/Simplifi-ED/grafana-apm-app/commit/7e4a81464553d64f72e03b1f24b88d44f24b2e0c))
* add bullet graph visualization for Core Web Vitals ([adccee2](https://github.com/Simplifi-ED/grafana-apm-app/commit/adccee23292a272c3e8fd50a840398e34f24a80a))
* add ClickHouse support and update configuration documentation ([099cb60](https://github.com/Simplifi-ED/grafana-apm-app/commit/099cb60bfeaec65d338082cf040575ffecef1f43))
* add dependencies panel to service overview tab ([bafb6ed](https://github.com/Simplifi-ED/grafana-apm-app/commit/bafb6edd009f9c3c8358cd18e2104b71c67ff28e))
* add Faro kind filter and fix exception-to-logs navigation ([a1f38ce](https://github.com/Simplifi-ED/grafana-apm-app/commit/a1f38ce470fd42c7c9819ac3dd8788e0765e68db))
* add favorite services ("My Apps") personalization ([c456df3](https://github.com/Simplifi-ED/grafana-apm-app/commit/c456df3e2770eadc2aca2a6939c840a1fca83750)), closes [#15](https://github.com/Simplifi-ED/grafana-apm-app/issues/15)
* add FrontendCapabilities contract with resolveCapabilities() ([2e30e4c](https://github.com/Simplifi-ED/grafana-apm-app/commit/2e30e4c2de5f30debe607bdf6866abf55cf6910d))
* add health summary section to service overview ([2f2d5ee](https://github.com/Simplifi-ED/grafana-apm-app/commit/2f2d5ee418fe29cae77485a518d45050508d0bde))
* add histogram-based frontend tab with hybrid Loki enrichment ([2ed47ed](https://github.com/Simplifi-ED/grafana-apm-app/commit/2ed47eddac2fb7444caccc55a262f2fcbdc570c3))
* add ingress alias support for on-prem caller visibility ([#44](https://github.com/Simplifi-ED/grafana-apm-app/issues/44)) ([1c8b962](https://github.com/Simplifi-ED/grafana-apm-app/commit/1c8b96254aa3a8e658d55fc42c38ba2d33d025a4))
* add local deployment tasks to mise ([4a18858](https://github.com/Simplifi-ED/grafana-apm-app/commit/4a18858de14849ca4e4ea370a3f0c2c38d7b649c))
* add mise e2e and e2e:ui tasks for local Playwright testing ([0bf1212](https://github.com/Simplifi-ED/grafana-apm-app/commit/0bf1212de795eff21365b8c0edf6cd26bc13d227))
* add mise run release task for automated releases ([a359259](https://github.com/Simplifi-ED/grafana-apm-app/commit/a35925953c3746cc15f8171b57387710fa936f56))
* add My Apps nav entry that redirects to favorites-filtered service list ([7d01c7a](https://github.com/Simplifi-ED/grafana-apm-app/commit/7d01c7a1d914e661db51e024300bbe65ae9830c0))
* add namespace overview page with shared components ([ab5f336](https://github.com/Simplifi-ED/grafana-apm-app/commit/ab5f336a9b217755dbd31b96e32dd1bd0f5774fc))
* add Ops Status Board for cross-namespace monitoring ([982c248](https://github.com/Simplifi-ED/grafana-apm-app/commit/982c248b5d91ed50005691902b3446103e20fdd0))
* add optional Pyroscope profiling support and update documentation ([c270b0e](https://github.com/Simplifi-ED/grafana-apm-app/commit/c270b0e250456db886f10a74b64a4ca1b9df91cc))
* add pageloads/sessions stats and fix exceptions table columns ([3f43fc1](https://github.com/Simplifi-ED/grafana-apm-app/commit/3f43fc16cc2583bdf21dccaf00c2d055b1e512d2))
* add panel descriptions to ServiceOverview ([f17203e](https://github.com/Simplifi-ED/grafana-apm-app/commit/f17203e66d90db10d1f0a718ef1ee12f54830761))
* add percentile accuracy help tooltip to server tab ([34587f8](https://github.com/Simplifi-ED/grafana-apm-app/commit/34587f82d9fc49254f07439aabb6312cfdd8857e))
* add RED time-series panels to dependency detail page ([af795a4](https://github.com/Simplifi-ED/grafana-apm-app/commit/af795a4be75197e079ed093af30cd318b3df018f))
* add search and pagination to namespace tables ([4bb0fa8](https://github.com/Simplifi-ED/grafana-apm-app/commit/4bb0fa8a14fe1a9c4882662ba739ffc5b9ecf527))
* add service topology graph to overview tab ([a3ab413](https://github.com/Simplifi-ED/grafana-apm-app/commit/a3ab41304e153d81eac964c28144cb136275e1ce))
* add Status Board view for TV displays ([#29](https://github.com/Simplifi-ED/grafana-apm-app/issues/29)) ([f0f3b48](https://github.com/Simplifi-ED/grafana-apm-app/commit/f0f3b48bf864d3cf6a0806bc5b3f03bb6aa49657))
* add support for label overrides in OTel pipelines ([#42](https://github.com/Simplifi-ED/grafana-apm-app/issues/42)) ([7fb5d3e](https://github.com/Simplifi-ED/grafana-apm-app/commit/7fb5d3e788d68aea9a80be55bddda378708971dc))
* add time range picker to service inventory page ([96f3d18](https://github.com/Simplifi-ED/grafana-apm-app/commit/96f3d189215a5dd2a8271273b84830c460dce90d))
* adjust lists limits, sanitize page urls, and relocate logs links ([14c7b4d](https://github.com/Simplifi-ED/grafana-apm-app/commit/14c7b4df57fe467f6efaad7018fc47af38ee736f))
* **alerting:** harden defaults= contract ([#77](https://github.com/Simplifi-ED/grafana-apm-app/issues/77)) + surface firing state ([#33](https://github.com/Simplifi-ED/grafana-apm-app/issues/33) Phase 1) ([#104](https://github.com/Simplifi-ED/grafana-apm-app/issues/104)) ([85b6a88](https://github.com/Simplifi-ED/grafana-apm-app/commit/85b6a88589913a8e617f12c051c9a8bd1b0db95d))
* **alerts:** new issues → Slack via a copy-pasteable PrometheusRule ([#177](https://github.com/Simplifi-ED/grafana-apm-app/issues/177)) ([257c88c](https://github.com/Simplifi-ED/grafana-apm-app/commit/257c88cfaa7e34b8d447a0e99d9b1ed134544e02))
* **alerts:** read-only firing-alert detail drawer ([#32](https://github.com/Simplifi-ED/grafana-apm-app/issues/32)) ([#110](https://github.com/Simplifi-ED/grafana-apm-app/issues/110)) ([2996d1e](https://github.com/Simplifi-ED/grafana-apm-app/commit/2996d1eaaa734606a1ad6544d1840aa047d7588c))
* align mise run all with CI, add pre-commit hook ([5cab2ab](https://github.com/Simplifi-ED/grafana-apm-app/commit/5cab2ab40475668505b46c2221e00518e0a1bd6a))
* auto-managed service account auth (v0.2.1) ([1daf48b](https://github.com/Simplifi-ED/grafana-apm-app/commit/1daf48be6dcd1cdd5d907ba9fa76ba8441ba244d))
* clickable endpoint rows navigate to filtered traces ([7921ee6](https://github.com/Simplifi-ED/grafana-apm-app/commit/7921ee6f0776b48506298e672a963ebb8ab4e7ef))
* clickable graph nodes with caller-count color coding ([e28a311](https://github.com/Simplifi-ED/grafana-apm-app/commit/e28a31194c74f8e7d80cea78bd7a52fcb3a93f74))
* clickable operations with traces drill-down and Operations tab ([f3ba24f](https://github.com/Simplifi-ED/grafana-apm-app/commit/f3ba24f20cfe6fbd39870c3259fd750aea685c48))
* collapse dormant nodes in service graph ([4e5c592](https://github.com/Simplifi-ED/grafana-apm-app/commit/4e5c592b1f0009a198626694528893857d8d2acd))
* configure private plugin signing with root URLs ([700a361](https://github.com/Simplifi-ED/grafana-apm-app/commit/700a36129288c0a0d001c589d3daa0c1647e9b52))
* consolidate context and impact layout in ExceptionDrawer ([333a171](https://github.com/Simplifi-ED/grafana-apm-app/commit/333a171676c9e24d7eeb42726d139b93ebd4e4b3))
* convert Makefile to mise task runner ([6988219](https://github.com/Simplifi-ED/grafana-apm-app/commit/6988219a5e9e7fcc8e114c5b313af24ebbd65009))
* copy team service topology as Mermaid graph ([c7547eb](https://github.com/Simplifi-ED/grafana-apm-app/commit/c7547eb86512d6fa9bac57e29c49d3feed67397f))
* custom service graph with React Flow + ELK.js ([2ac0ae9](https://github.com/Simplifi-ED/grafana-apm-app/commit/2ac0ae912ee5fb170dfc189ddb6f6efa96bb4488))
* **database:** Phase 2 — Top queries (normalized, Tempo-sourced) ([#119](https://github.com/Simplifi-ED/grafana-apm-app/issues/119)) ([#124](https://github.com/Simplifi-ED/grafana-apm-app/issues/124)) ([a6c427d](https://github.com/Simplifi-ED/grafana-apm-app/commit/a6c427d2d10e53802ab99fef75da5688d4b248d5))
* datasource config with dropdown selectors and auto-detection ([50fe557](https://github.com/Simplifi-ED/grafana-apm-app/commit/50fe55790275f6acddefa6aa532f564021fb2a36))
* deep linking for service tabs via ?tab= URL param ([db2e39e](https://github.com/Simplifi-ED/grafana-apm-app/commit/db2e39eac1320633d7fde8bba02fb2fcdef0d69e))
* distinguish 'no instrumentation' from 'no recent data' ([48f6adf](https://github.com/Simplifi-ED/grafana-apm-app/commit/48f6adfcc15ff00f8fd8f198748db92b816a8782))
* enable ELK MULTI_EDGE wrapping for large topology graphs ([1c632f8](https://github.com/Simplifi-ED/grafana-apm-app/commit/1c632f8fb225811378107cb3505be2e6ef540ee7))
* enhance ClickHouse integration and update documentation ([e29fd9e](https://github.com/Simplifi-ED/grafana-apm-app/commit/e29fd9eec92d75a9edc6775ed3ce581b0774c71a))
* enhance ClickHouse integration for alert templates and log/traces search ([a1e28e6](https://github.com/Simplifi-ED/grafana-apm-app/commit/a1e28e61e2cf17cd655b180cc83d83f997822c7c))
* enhance ClickHouse integration with improved service metrics and custom metrics discovery ([8d0f4a2](https://github.com/Simplifi-ED/grafana-apm-app/commit/8d0f4a23f2b282d2a74c80adb109f9ac637ae04b))
* enhance ClickStack APM integration with ClickHouse support ([d8ff872](https://github.com/Simplifi-ED/grafana-apm-app/commit/d8ff87210eb68d282ec67f531ef80645245c4e44))
* enhance ClickStack integration with histogram-based latency metrics and improved operations parsing ([e5654db](https://github.com/Simplifi-ED/grafana-apm-app/commit/e5654db936ca4ee0e2a60704d00d18507c6d39b5))
* enhance ClickStack log and trace search functionality ([05d2fea](https://github.com/Simplifi-ED/grafana-apm-app/commit/05d2feafa22be7823e322771cf597274b3f60a80))
* enhance ClickStack log and trace search functionality ([d560bab](https://github.com/Simplifi-ED/grafana-apm-app/commit/d560babd04dace14177bca8762ab3c345c0ef0d8))
* enhance dependency mapping with spanmetrics supplement ([57da0fc](https://github.com/Simplifi-ED/grafana-apm-app/commit/57da0fcb65f0c56a19b089011d6d50cc27ca066c))
* enhance framework detection and SQL queries for ClickStack integration ([f9b7fad](https://github.com/Simplifi-ED/grafana-apm-app/commit/f9b7fad1edc71e57b6f78b928f50a6716ae14909))
* enhance jobs namespace handling in query logic ([c288eca](https://github.com/Simplifi-ED/grafana-apm-app/commit/c288ecaded386f35ab0fadd7ed8db7c5d3e710c6))
* enhance Playwright configuration and CI workflow for improved testing ([c4ab213](https://github.com/Simplifi-ED/grafana-apm-app/commit/c4ab213745ffd6a14763deb7a1bf44bf3473d1e3))
* enhance service summary fetching with series data and filtering ([ded6ebf](https://github.com/Simplifi-ED/grafana-apm-app/commit/ded6ebfedf8907979ea25b9a0b1b6341ef0377a7))
* enhanced namespace page with health indicators and attention section ([e6d340d](https://github.com/Simplifi-ED/grafana-apm-app/commit/e6d340d0d8ccdf89820e10eaa2070ecafc629991))
* enrich database nodes in service map with db_system and db_name ([597b817](https://github.com/Simplifi-ED/grafana-apm-app/commit/597b8179fc172e0b56b8866a3f51fdd042a43a3e))
* environment dropdown in config and detect environments from data ([b93172e](https://github.com/Simplifi-ED/grafana-apm-app/commit/b93172ed89826b922d6f442512b73ff6cdd800bd))
* environment-aware dependencies and improved env handling ([8d5aa30](https://github.com/Simplifi-ED/grafana-apm-app/commit/8d5aa308ed6c2b0b99f7a04d7c81e8127cf6ceab))
* environment-aware metrics datasource facade ([6d9abf3](https://github.com/Simplifi-ED/grafana-apm-app/commit/6d9abf38323972adb332a896859601ac4745dc91))
* exception drawer breadcrumbs, user context, and impact aggregation ([b7fc905](https://github.com/Simplifi-ED/grafana-apm-app/commit/b7fc905a75e73a6a21714d5e2b1438ee0da05f4a))
* expand runtime tab with container resources, Go runtime, JVM pools, GC types ([2f16fd1](https://github.com/Simplifi-ED/grafana-apm-app/commit/2f16fd19484278d535a223bf72e124842408a104))
* extract vital values from Loki for bullet graph display ([bc63d84](https://github.com/Simplifi-ED/grafana-apm-app/commit/bc63d842d838fd14c5e1fc963b05c5489646afb9))
* frontend APM exception explorer and loki enrichment ([8f7a588](https://github.com/Simplifi-ED/grafana-apm-app/commit/8f7a588c74e1e7e9d8801e6e0e4dd886e3c04260))
* frontend session timeline improvements (grouping, session switching, and event attributes) ([f8537eb](https://github.com/Simplifi-ED/grafana-apm-app/commit/f8537ebf706f6ee166f3cc3dfd5fc5fa8b5faccd))
* **frontend:** add browser filter dropdown to Frontend tab ([e24b85e](https://github.com/Simplifi-ED/grafana-apm-app/commit/e24b85e32c80a9b0bfb965c6bc1d31ac65271fb8))
* **frontend:** detect minified stacks + source-map doctor ([#60](https://github.com/Simplifi-ED/grafana-apm-app/issues/60)) ([#103](https://github.com/Simplifi-ED/grafana-apm-app/issues/103)) ([c402ffa](https://github.com/Simplifi-ED/grafana-apm-app/commit/c402ffa248f496f1a4cbcde688352cc14c2f95cc))
* **frontend:** improve Frontend Tab with correct aggregation and new panels ([5c67156](https://github.com/Simplifi-ED/grafana-apm-app/commit/5c67156ba5d3b60a59aece6507b67095cbdaf6ba))
* **frontend:** improve table readability and dashboard layout ([3cfac59](https://github.com/Simplifi-ED/grafana-apm-app/commit/3cfac5903a83480180cd580fd73932cb67b0ad4f))
* graph UI enhancements (fullscreen toggle, mermaid export, ELK fix) ([d0b9b33](https://github.com/Simplifi-ED/grafana-apm-app/commit/d0b9b334e8741a1d635b83a240c8939002f3376e))
* group services by environment when namespace is filtered ([67741db](https://github.com/Simplifi-ED/grafana-apm-app/commit/67741dbd44c42e09ac96637d2aea1ca92dcee676))
* hub detection with rate-weighted pruning and name normalization ([105cf92](https://github.com/Simplifi-ED/grafana-apm-app/commit/105cf92fc1d5f43ca598b0864fcdc247f559a7ba))
* implement session replay functionality for HyperDX ([bbc966b](https://github.com/Simplifi-ED/grafana-apm-app/commit/bbc966bc33c726679f3df865ab90b495c4c28d73))
* improve ClickHouse error handling and health checks ([8ffdbf5](https://github.com/Simplifi-ED/grafana-apm-app/commit/8ffdbf586555f1516c91f8b16ff2eecbade4cff2))
* integrate unused Mimir pipeline metrics into Frontend tab ([8f4c0d1](https://github.com/Simplifi-ED/grafana-apm-app/commit/8f4c0d12cc93d3aacf522625c90a27196978b2d4))
* **issues:** richer context for backend log occurrences ([#118](https://github.com/Simplifi-ED/grafana-apm-app/issues/118)) ([b77673b](https://github.com/Simplifi-ED/grafana-apm-app/commit/b77673bf6c8c4e0438fc7fd81b13e731ac0f100d))
* **issues:** server-issue drawer parity — open the exception drawer for backend errors ([#84](https://github.com/Simplifi-ED/grafana-apm-app/issues/84) Phase 1) ([#93](https://github.com/Simplifi-ED/grafana-apm-app/issues/93)) ([458950a](https://github.com/Simplifi-ED/grafana-apm-app/commit/458950a34d0d3e5512781fb17359ef3d244eb0f2))
* leverage production OTel dimensions and exemplars ([df35788](https://github.com/Simplifi-ED/grafana-apm-app/commit/df35788af1b217ca35495e95224a61f1dd7ef7d1))
* link connected services to service detail page ([7451ff7](https://github.com/Simplifi-ED/grafana-apm-app/commit/7451ff76b7ee7b4046d6a1206ac82f927958700d))
* link service-type dependencies to service overview page ([077a689](https://github.com/Simplifi-ED/grafana-apm-app/commit/077a689d43c59b155b374e64ccd17b97f1544fff))
* **logs:** add pod filter, severity stream labels, and faro toggle to Logs tab ([2e253bf](https://github.com/Simplifi-ED/grafana-apm-app/commit/2e253bfa0589b294ba66e855f7c98ba08767d450))
* make selected session ID in Exception Drawer stable and shareable in the URL ([0c334f0](https://github.com/Simplifi-ED/grafana-apm-app/commit/0c334f0f25df7d64b1ecffbf749be31e7e882fa7))
* multi-hop service topology with depth control (1–3 hops) ([81d0cd1](https://github.com/Simplifi-ED/grafana-apm-app/commit/81d0cd157d6fad9f5c645c1fa2fba273b4c4b7b4))
* multi-select environment filter on all pages ([1f8e8f6](https://github.com/Simplifi-ED/grafana-apm-app/commit/1f8e8f62b10f77ebe9eac5d9f19e7eae33f7b496))
* Nais APM Legend-edition ([#71](https://github.com/Simplifi-ED/grafana-apm-app/issues/71)) ([2e48428](https://github.com/Simplifi-ED/grafana-apm-app/commit/2e484287adc60ea1301318f342ca2dd68c2c15ac))
* **naissync:** prune stale deploy annotations ([#173](https://github.com/Simplifi-ED/grafana-apm-app/issues/173)) ([dba7c66](https://github.com/Simplifi-ED/grafana-apm-app/commit/dba7c66c526daed24e5d4ea2697eb1c0890a5e47)), closes [#128](https://github.com/Simplifi-ED/grafana-apm-app/issues/128)
* namespace/team filter as multi-select on service list page ([86d21e1](https://github.com/Simplifi-ED/grafana-apm-app/commit/86d21e1fa7036abc9694495e3cf4f5dceb866e93))
* **ops-board:** make watchlist globally editable by any user ([8c81561](https://github.com/Simplifi-ED/grafana-apm-app/commit/8c81561988f1366bc8d91be82260122c22ebd265))
* **ops-board:** use user favorites instead of admin watchlist ([516654f](https://github.com/Simplifi-ED/grafana-apm-app/commit/516654f11df2300c976225b0ded82e40c3cc60e5))
* **ops:** add inline service picker ([b179297](https://github.com/Simplifi-ED/grafana-apm-app/commit/b1792971a5d44cb517646246a18f0016b23ae4cc))
* per-hop overflow collapsing for multi-hop topology ([df76270](https://github.com/Simplifi-ED/grafana-apm-app/commit/df76270094875f1b2fff62724b6c0448ba5589f9))
* persist all filter/sort state to URL query parameters ([5fdac3b](https://github.com/Simplifi-ED/grafana-apm-app/commit/5fdac3b5fe68281eb1514c381d78a333228153b0))
* prefer page_id over page_url for per-page performance grouping ([bc5b89c](https://github.com/Simplifi-ED/grafana-apm-app/commit/bc5b89ce95af005cacc292de6a755b1fa765419c)), closes [#41](https://github.com/Simplifi-ED/grafana-apm-app/issues/41)
* refactor API authentication and improve service graph layout ([05f94b3](https://github.com/Simplifi-ED/grafana-apm-app/commit/05f94b3f8c2487a90850bc58c10fa8c4eb497a0a))
* remove global Service Map page ([6e71ada](https://github.com/Simplifi-ED/grafana-apm-app/commit/6e71ada07f8210aac2cf92312628ae5ef79208c4))
* rich rendering for timeline breadcrumbs ([2d63ec5](https://github.com/Simplifi-ED/grafana-apm-app/commit/2d63ec5ad748175e1dc511ae846033ee252ff3c8))
* **scorecard:** make every failing readiness check one click to fix ([#176](https://github.com/Simplifi-ED/grafana-apm-app/issues/176)) ([7abe3ad](https://github.com/Simplifi-ED/grafana-apm-app/commit/7abe3ad4db8cd0540f2f8cf97b0c09b6e0cad11d))
* **seed:** deterministic synthetic Faro telemetry generator ([#135](https://github.com/Simplifi-ED/grafana-apm-app/issues/135)) ([dcf3c97](https://github.com/Simplifi-ED/grafana-apm-app/commit/dcf3c97c6472ccc62db13929b4395bdae5a98961))
* server-side namespace filtering for service map and dependencies ([1c23ff8](https://github.com/Simplifi-ED/grafana-apm-app/commit/1c23ff8c9d6071d00ae87076adb572913ff35377))
* service account token auth and enhanced config page ([394b10e](https://github.com/Simplifi-ED/grafana-apm-app/commit/394b10ef880a9c25b6984847b647abbab76a54fc))
* show alert source label (Mimir) in namespace alerts header ([c5ac071](https://github.com/Simplifi-ED/grafana-apm-app/commit/c5ac0719f65fe5c7d0f45c67a652d5ca1da422c5))
* show Callers and Dependencies sections on Dependencies tab ([bca2987](https://github.com/Simplifi-ED/grafana-apm-app/commit/bca2987f9ba67a438f2a6d66061482bfc88ba1db))
* show Mimir alert rules on namespace page ([0f8f9d0](https://github.com/Simplifi-ED/grafana-apm-app/commit/0f8f9d077093aad4ff98fec4262b7d6c4f0b8510))
* sidecar-aware service display and improved framework detection ([df8a1ff](https://github.com/Simplifi-ED/grafana-apm-app/commit/df8a1ff80ece01648d18c61fe1309141cfd44051))
* simplify Faro resource and navigation events in timeline ([8d9c494](https://github.com/Simplifi-ED/grafana-apm-app/commit/8d9c4949296cedbf9788423f376906c5f6c4bcee))
* **status-board:** multi-select environment filter ([2d1fe89](https://github.com/Simplifi-ED/grafana-apm-app/commit/2d1fe89f776da5140d0ae73a1730ead3fa54a8fb))
* **status-board:** viewport-filling layout with auto-pagination ([1549964](https://github.com/Simplifi-ED/grafana-apm-app/commit/1549964c7912c7d6d2007011190688a5e44164d0))
* support Alloy Faro pipeline metrics in frontend tab ([a90dab2](https://github.com/Simplifi-ED/grafana-apm-app/commit/a90dab2a8e6b9f8fc76c461015af2088190a1b3c))
* support centralized Loki with cluster-scoped queries ([831628c](https://github.com/Simplifi-ED/grafana-apm-app/commit/831628c5965431767174e92fe3b9701ba8104a72))
* support multi-environment filtering in backend queries ([d175d04](https://github.com/Simplifi-ED/grafana-apm-app/commit/d175d04b5d769327396e04c270d56be891cb6079))
* sync favorites to Grafana user storage for cross-device persistence ([0423a29](https://github.com/Simplifi-ED/grafana-apm-app/commit/0423a29932317f547c334efef910e2d03b1fc4fc))
* transition from nais-apm-app to clickhouse-apm-app branding and configuration ([1320dfd](https://github.com/Simplifi-ED/grafana-apm-app/commit/1320dfd6089e1b928d38b284cee280c4ad3ec77d))
* update framework badge colors and improve framework detection logic ([eac9d16](https://github.com/Simplifi-ED/grafana-apm-app/commit/eac9d166c4b717fc25f7d486d57948453da4028a))
* update GraphQL span emitter and deployment configuration ([888b224](https://github.com/Simplifi-ED/grafana-apm-app/commit/888b2248d57ddab46f897334438dce71363179b9))
* update Pyroscope profiling integration and documentation ([419bd63](https://github.com/Simplifi-ED/grafana-apm-app/commit/419bd635171abda5d0c71e82034350ca6b795e09))
* use full-height canvas layout for all service pages ([8813d02](https://github.com/Simplifi-ED/grafana-apm-app/commit/8813d028942e9566a74a994a77c01b4cbf99cf59))
* use service graph db_system/messaging_system directly, drop spanmetrics enrichment ([7129b5f](https://github.com/Simplifi-ED/grafana-apm-app/commit/7129b5ff4cd46ea70b2845a620c3327b63d74455)), closes [#23](https://github.com/Simplifi-ED/grafana-apm-app/issues/23)
* use telemetry_sdk_language as fallback for framework detection ([bffb2cf](https://github.com/Simplifi-ED/grafana-apm-app/commit/bffb2cf78aec4778a091fa4b826996f07b323562)), closes [#20](https://github.com/Simplifi-ED/grafana-apm-app/issues/20)


### Bug Fixes

* add .npmrc with auto-install-peers and regenerate lockfile ([8804fa2](https://github.com/Simplifi-ED/grafana-apm-app/commit/8804fa2838e1ebaaf635c7a4e19f0766906c02d0))
* add @swc/register to fix webpack build ([a25589c](https://github.com/Simplifi-ED/grafana-apm-app/commit/a25589c53fb9479afe7ac0942e80e22bdb029258))
* add debugging for Grafana 13 e2e config test failure ([86b4fff](https://github.com/Simplifi-ED/grafana-apm-app/commit/86b4fffc28939cae95dd1966cd9fee90392366eb))
* add environment filtering to runtime, service map, and frontend Mimir queries ([de7fe63](https://github.com/Simplifi-ED/grafana-apm-app/commit/de7fe63026552d0f0fbc6bafbf16b411d407e697))
* add loki_process_custom_ prefix to histogram metric names ([253bb8b](https://github.com/Simplifi-ED/grafana-apm-app/commit/253bb8bc8d8724860c4ba610ff00f539432f944f))
* add spanmetrics fallback for topology graph in environments without service graph metrics ([ff5fb81](https://github.com/Simplifi-ED/grafana-apm-app/commit/ff5fb81df558339efe7dc166c6ad8da6093d65f7))
* address design philosophy audit findings ([722e8ce](https://github.com/Simplifi-ED/grafana-apm-app/commit/722e8ce9e4ee5a4a529c2331cc6f4d3be2ac1f95))
* address multi-select env filter review findings ([2da5bf7](https://github.com/Simplifi-ED/grafana-apm-app/commit/2da5bf7517fa238d7f006c0d4aa67aa49ab00df8))
* alerts endpoint compatibility with NAIS Mimir ([57d4780](https://github.com/Simplifi-ED/grafana-apm-app/commit/57d47802d1e228bfee1697e0619117e6df33a96a))
* align counter names with actual pipeline deployment ([2890ab2](https://github.com/Simplifi-ED/grafana-apm-app/commit/2890ab27679260f99aa4807c6fc4015ea6439f82))
* align histogram config with actual Alloy pipeline implementation ([36f54d5](https://github.com/Simplifi-ED/grafana-apm-app/commit/36f54d5fd4b63ea2a21f56758ce064a3c31fa3df))
* align impact column bar+label to the right in dependencies tab ([4c34a2b](https://github.com/Simplifi-ED/grafana-apm-app/commit/4c34a2bfeacc28454ff967900332df3d8a729326))
* always use Canvas layout for service detail pages ([e43701f](https://github.com/Simplifi-ED/grafana-apm-app/commit/e43701f808cbee961ed7d773b521314858952447))
* back button visible by default, remove text-only fill ([1724155](https://github.com/Simplifi-ED/grafana-apm-app/commit/17241558a0b289db1917036ea83fbef61d9f956d))
* Browser Volume pie chart — use range query for legend labels ([ef0e3cd](https://github.com/Simplifi-ED/grafana-apm-app/commit/ef0e3cd154b83cb0a754b5d68a3bff8cae4555db))
* **build:** drop tsconfig baseUrl ahead of the TypeScript 7 bump ([#179](https://github.com/Simplifi-ED/grafana-apm-app/issues/179)) ([a6399fc](https://github.com/Simplifi-ED/grafana-apm-app/commit/a6399fc045519cb992cd6ceb751c01c5c8ebffc5))
* **build:** migrate TypeScript config for version 7 ([#155](https://github.com/Simplifi-ED/grafana-apm-app/issues/155)) ([fa27c8d](https://github.com/Simplifi-ED/grafana-apm-app/commit/fa27c8dfdd98695571f966f99186f81e7b1304ff))
* **ci:** e2e only starts Grafana, not full stack ([062c89d](https://github.com/Simplifi-ED/grafana-apm-app/commit/062c89d1e8a1cc54132b3d61c4e0616a33710f0d))
* **ci:** generate release notes from CHANGELOG.md ([03d8bd8](https://github.com/Simplifi-ED/grafana-apm-app/commit/03d8bd88cd91c3b8eec15201ba5565e30a5e6ac5))
* **ci:** publish release instead of leaving as draft ([374763d](https://github.com/Simplifi-ED/grafana-apm-app/commit/374763db71547328e4d384b6b367faba0e06e388))
* **ci:** strip plugin signature for E2E tests ([5e96abc](https://github.com/Simplifi-ED/grafana-apm-app/commit/5e96abc9a82b3b3b5be73645dbf5eced9ea395b7))
* **ci:** use env context instead of secrets in if condition ([13a3465](https://github.com/Simplifi-ED/grafana-apm-app/commit/13a3465d6da1a658b58c694b87b1a4b7d880d2fb))
* clarify release commit titles ([#200](https://github.com/Simplifi-ED/grafana-apm-app/issues/200)) ([509c644](https://github.com/Simplifi-ED/grafana-apm-app/commit/509c644b435b66066d8a96e02d335f10199cc6d8))
* **clickstack:** polish sessions enrich, logs identity, scorecard jobs ([e2a223e](https://github.com/Simplifi-ED/grafana-apm-app/commit/e2a223e681607dec2f8c144154e7592f17cef594))
* client-side filtering on Dependencies and ServiceInventory pages ([3437478](https://github.com/Simplifi-ED/grafana-apm-app/commit/3437478704868b1dda6105ac133bcaef05a7b89b))
* command palette integration and changelog update ([def3cde](https://github.com/Simplifi-ED/grafana-apm-app/commit/def3cde97a685801834f894623f336df868c2939))
* compact Faro badge to icon-only, badges on same row ([10316e0](https://github.com/Simplifi-ED/grafana-apm-app/commit/10316e0b35c3747f7430b39e35db65b954bb805e))
* **config:** allow clearing the service account token + de-bloat the page ([#92](https://github.com/Simplifi-ED/grafana-apm-app/issues/92)) ([b6afbc2](https://github.com/Simplifi-ED/grafana-apm-app/commit/b6afbc2fd57a5cde9824a3a96b1e36e4e631aba1))
* consistent badge labels and Faro badge spacing ([d843cff](https://github.com/Simplifi-ED/grafana-apm-app/commit/d843cffca0480fd84d6e05ced8719d3239494baa))
* consistent error rate calculation and input sanitization ([2b2abb7](https://github.com/Simplifi-ED/grafana-apm-app/commit/2b2abb747d8b9973830feda323352a9ceb4b0ce8))
* consistent text truncation across all tables ([dc9f504](https://github.com/Simplifi-ED/grafana-apm-app/commit/dc9f50457b66e8c69a2fecf4bdf981ee0088bc43))
* constrain Scenes panel height on Canvas layout ([d50c792](https://github.com/Simplifi-ED/grafana-apm-app/commit/d50c7925ba0abd9f9bbb12c8f13699fd93b29f08))
* constrain Scenes panel height with wrapper divs ([e6ed2a8](https://github.com/Simplifi-ED/grafana-apm-app/commit/e6ed2a81fd5a06340022ad69d85cd151ce0bd1e0))
* consume hasLoki from backend to guard Loki-dependent panels ([7c3f15b](https://github.com/Simplifi-ED/grafana-apm-app/commit/7c3f15b191488cbb24496e6784059b90b8b748f3))
* convert README relative links to absolute URLs ([8aa2247](https://github.com/Simplifi-ED/grafana-apm-app/commit/8aa22473a7da99091b876467abe3aeb3894e032f))
* correct exception links to use ?tab=logs instead of /logs path ([72ab84a](https://github.com/Simplifi-ED/grafana-apm-app/commit/72ab84a746595cad3f6acc13412b9a2bd7dee475))
* correct service map error rate calculation ([8d32ced](https://github.com/Simplifi-ED/grafana-apm-app/commit/8d32ced9edbc001fabc8007d6666446d8e8e838d))
* critical bugs from code review — navigation, auth, regex, useFetch ([589f13c](https://github.com/Simplifi-ED/grafana-apm-app/commit/589f13cf1d53013387957e6032a4ccaaaf4723aa))
* CWV Rating as stat panel, Browser Volume labels, 1h default ([c95a171](https://github.com/Simplifi-ED/grafana-apm-app/commit/c95a171900abadf6e7d9eb47fcbafd3acf5164af))
* CWV Rating uses histogram buckets, Browser Volume uses Loki ([743a1aa](https://github.com/Simplifi-ED/grafana-apm-app/commit/743a1aaa58280322030cabb4049e35c4ebe26fd6))
* deduplicate cross-cluster alerts and improve namespace page UI ([9c30bbf](https://github.com/Simplifi-ED/grafana-apm-app/commit/9c30bbf633c45fd5faf07f9c6cc2f53164fadba5))
* dependency detail for external services (address normalization, dedup, P95) ([b08aec0](https://github.com/Simplifi-ED/grafana-apm-app/commit/b08aec08f94ad9b6880198956c92420feddd53dc))
* dependency detail uses regex matching for server label ([d0b9336](https://github.com/Simplifi-ED/grafana-apm-app/commit/d0b933634e222034c2df8dd4829dd95448657e5d))
* dependency error rate displayed as 10000% instead of 100% ([88f62d4](https://github.com/Simplifi-ED/grafana-apm-app/commit/88f62d46c93f2d57845e1dd4f9d9c7009431413a))
* **deps:** patch vulnerable transitive dependencies ([1e112e6](https://github.com/Simplifi-ED/grafana-apm-app/commit/1e112e60a3ce7b893c67be610651e6c742b315aa))
* **deps:** pin transitive dependencies flagged by Dependabot ([#178](https://github.com/Simplifi-ED/grafana-apm-app/issues/178)) ([8cfbc1f](https://github.com/Simplifi-ED/grafana-apm-app/commit/8cfbc1f34881815ca30423c831160215c58639aa))
* discover services from all span kinds, not just SERVER ([b017523](https://github.com/Simplifi-ED/grafana-apm-app/commit/b0175237d51a03ec0986ff711a3fcb78ce0b099d))
* dismiss Grafana 13 'What's new' modal in e2e config tests ([66b4805](https://github.com/Simplifi-ED/grafana-apm-app/commit/66b48059a59ef949d53715dc36d1e61755bc5e49))
* dismiss Grafana 13 What's New carousel overlay in e2e tests ([27201d0](https://github.com/Simplifi-ED/grafana-apm-app/commit/27201d07e49c32537ce6b0cee4705d36aa5657f2))
* double-escape PromQL regex to fix dependency detail queries ([b4d1c25](https://github.com/Simplifi-ED/grafana-apm-app/commit/b4d1c2573df81ca69d2804baf9040f60bffeadd4))
* **e2e:** add explicit route redirect and resilient nav locators ([08c0d82](https://github.com/Simplifi-ED/grafana-apm-app/commit/08c0d82dd9c0b9e7d4c4b4822e6ef4616eda1f4b))
* **e2e:** make remaining tests resilient to Grafana version differences ([a6ec960](https://github.com/Simplifi-ED/grafana-apm-app/commit/a6ec960404c97025d19a1b55652fb1b88b7fae6b))
* **e2e:** replace networkidle with load state wait ([e5a1c8b](https://github.com/Simplifi-ED/grafana-apm-app/commit/e5a1c8b53031059b7a35f62d49c6e7a8af98bee9))
* **e2e:** use .or() locator chains for cross-version compatibility ([e3e391a](https://github.com/Simplifi-ED/grafana-apm-app/commit/e3e391ad93819abc35d68ca1a35a8aa5c6431cd1))
* env dropdown, multi-env rows, consistent back button, badge styling ([11f2fd0](https://github.com/Simplifi-ED/grafana-apm-app/commit/11f2fd0b9d1b9c1013a23ff3c410caeed7f25302))
* equalize stat panel widths (15% each), nav type gets ~55% ([83ca4cf](https://github.com/Simplifi-ED/grafana-apm-app/commit/83ca4cf9dee0aa02e151315cad90b33d93285f50))
* escape dots in dependency names for PromQL regex matchers ([b689795](https://github.com/Simplifi-ED/grafana-apm-app/commit/b689795f9e6196a4fcb8eaa18f22e9811feb2f81))
* escape service name and search in TraceQL queries ([ac14465](https://github.com/Simplifi-ED/grafana-apm-app/commit/ac14465c68eefe693b5e6dd4c9fb0b6257da4c9d))
* exclude internal services from dependencies, classify hostnames as external ([2a05eff](https://github.com/Simplifi-ED/grafana-apm-app/commit/2a05eff4ffa0db0f64ebbafb47a10100f3f2ab3e))
* exclude Kafka consumer services from dependencies list ([54ffcb2](https://github.com/Simplifi-ED/grafana-apm-app/commit/54ffcb26b090082bba7efa6cf47166db29137fc3))
* exclude sidecar containers from Go framework detection ([3b0da8e](https://github.com/Simplifi-ED/grafana-apm-app/commit/3b0da8ec733a9bdbee0f5f5acb9ebfed3eed0f67))
* expand console error panel Loki query to capture Faro exception console.errors ([e83d058](https://github.com/Simplifi-ED/grafana-apm-app/commit/e83d058e196280d802359cd9b9bde2e543ec416b))
* fall through to Loki when histogram vitals are all NaN ([1ba3c02](https://github.com/Simplifi-ED/grafana-apm-app/commit/1ba3c026cecaf760aea855795df57750b6e43ded))
* fetch plugin jsonData from API when config.apps is empty ([bfda960](https://github.com/Simplifi-ED/grafana-apm-app/commit/bfda960c67df1f6664c4946d67c6a9fcfce81719))
* filter dependencies and connected services by selected environment ([af4e936](https://github.com/Simplifi-ED/grafana-apm-app/commit/af4e9367a30e5bcf6e0b1f8b904e7cb7c49e3d1e))
* filter logs when environments share Loki ([#195](https://github.com/Simplifi-ED/grafana-apm-app/issues/195)) ([fde59fb](https://github.com/Simplifi-ED/grafana-apm-app/commit/fde59fbc6ade98b35a49a0dc7d8fd653bcb0eba5))
* filter service graph edges by environment ([03f55a8](https://github.com/Simplifi-ED/grafana-apm-app/commit/03f55a8314f49c82dec90df0fb23eb072ffef8c8))
* framework badge styling and empty namespace navigation ([456e017](https://github.com/Simplifi-ED/grafana-apm-app/commit/456e017781a1d04138d21e08af10513226738620))
* **frontend-tab:** prefer Loki over Alloy for accurate vitals data ([7cf788a](https://github.com/Simplifi-ED/grafana-apm-app/commit/7cf788a856e2c4d176dc24b3cd48b426290ed5fa))
* **frontend:** address code review findings in Frontend Tab ([e29d4ea](https://github.com/Simplifi-ED/grafana-apm-app/commit/e29d4ead8780f2fae87e354e31de1bcef4cb765d))
* **frontend:** separate unwrap from count_over_time in Loki queries ([a710405](https://github.com/Simplifi-ED/grafana-apm-app/commit/a710405e17af6f43bb68e0a559316675d60beb6e))
* **grafana:** support Grafana 13 local development ([#152](https://github.com/Simplifi-ED/grafana-apm-app/issues/152)) ([dbd7dc3](https://github.com/Simplifi-ED/grafana-apm-app/commit/dbd7dc32df12074d9f1e5f485c544fbfaaac3a93))
* handle NaN from histogram_quantile on fresh data ([26b5368](https://github.com/Simplifi-ED/grafana-apm-app/commit/26b53689195d98f367612d82d82257d784747f42))
* handle non-JSON logs gracefully in LogQL pipeline ([879bf60](https://github.com/Simplifi-ED/grafana-apm-app/commit/879bf606fc6a41a331e8281ec0d226c6ba57af6f))
* handle services with empty namespace in routing ([f0cf476](https://github.com/Simplifi-ED/grafana-apm-app/commit/f0cf476f7c95eca31a1425cc1aa5c9e5ffebc83c))
* handle unknown alert states and invalid dates gracefully ([9e28d27](https://github.com/Simplifi-ED/grafana-apm-app/commit/9e28d27c460281bc576f070db461f0d99d177754))
* handle wildcard namespace in service detail API URLs ([8037409](https://github.com/Simplifi-ED/grafana-apm-app/commit/80374094c085cf9ee64c49abca9f81ee92b9818b))
* hide namespace column when namespace filter is active ([1707f92](https://github.com/Simplifi-ED/grafana-apm-app/commit/1707f924d89c1ac09d609c8eacd2f5910d4fb886))
* hide sidecars, fix empty topology, remove 20-service cap ([98d0784](https://github.com/Simplifi-ED/grafana-apm-app/commit/98d0784282afe7ce4c552f636c6e6a5ce82ece37))
* improve service graph rendering — compact nodes, hidden handles, better spacing ([6354a84](https://github.com/Simplifi-ED/grafana-apm-app/commit/6354a840092e205c05f39bb5349b9752ae11defa))
* include external domains in global dependencies list ([a993d8e](https://github.com/Simplifi-ED/grafana-apm-app/commit/a993d8e80807c60efa85fdf54a0da4e4e978ff37))
* increase panel heights and add overflow hidden to prevent overlap ([69d4be3](https://github.com/Simplifi-ED/grafana-apm-app/commit/69d4be383f6b8fa2a32477bea3efb26fffefc8cb))
* increase panel wrapper heights to prevent axis label clipping ([d11bea5](https://github.com/Simplifi-ED/grafana-apm-app/commit/d11bea56f0442bc22b52383aef71955aedb1100d))
* **issues:** anchor the server-occurrence scan so drawer details match the list ([#157](https://github.com/Simplifi-ED/grafana-apm-app/issues/157)) ([0bade9b](https://github.com/Simplifi-ED/grafana-apm-app/commit/0bade9b7cd886489955439a32f8c820a080a0b99))
* **issues:** collapse stdout copy of an OTLP-exported exception ([#171](https://github.com/Simplifi-ED/grafana-apm-app/issues/171)) ([1c3bab0](https://github.com/Simplifi-ED/grafana-apm-app/commit/1c3bab0068b39ed8056368def5619ad93ca876eb))
* let Error column auto-expand to fill table width ([2a08cd5](https://github.com/Simplifi-ED/grafana-apm-app/commit/2a08cd551fc0691cb7c18c55694e535f20e2fa79))
* **logs:** expand severity filter to match all detected_level case variants ([65e9724](https://github.com/Simplifi-ED/grafana-apm-app/commit/65e97245fa1794fb3a35793d70b270c76664b8bd))
* **logs:** move severity filter from stream selector to pipeline ([d9b2a2b](https://github.com/Simplifi-ED/grafana-apm-app/commit/d9b2a2bffa7b752abf3d45341a5aa7b1af30cf2b))
* make CI signing non-fatal, add local plugin validation ([8ba2e8f](https://github.com/Simplifi-ED/grafana-apm-app/commit/8ba2e8f72ec4e44cb9f454fdf791aa3fe1843732))
* make favorites sync remote-authoritative, fix deletion propagation ([21c90b7](https://github.com/Simplifi-ED/grafana-apm-app/commit/21c90b7f596fb5a3b563c990565d81c361ec7eae))
* make mise tasks sandbox-resilient with silent output ([af72884](https://github.com/Simplifi-ED/grafana-apm-app/commit/af72884156179825645e259bbd5a4ddd9b08204d))
* make toolbar filters responsive with flex-grow ([47283e3](https://github.com/Simplifi-ED/grafana-apm-app/commit/47283e38d10d5b72111b31f41c9b98fbec98c871))
* match Faro badge height to framework badges ([554e753](https://github.com/Simplifi-ED/grafana-apm-app/commit/554e7537e792db0e8c1f7b8f6a3ff839ea56d8c0))
* move bullet graphs inside Scene and fix rating layout shift ([305dc87](https://github.com/Simplifi-ED/grafana-apm-app/commit/305dc87f54cb92e3479aef06a7dbb15e2529ad44))
* move duration filter inside TraceQL span selector ([0b28596](https://github.com/Simplifi-ED/grafana-apm-app/commit/0b285968d2bc7188bc16aa08bc82edb521de11bc))
* move framework/Faro badges to dedicated Type column ([9de63f1](https://github.com/Simplifi-ED/grafana-apm-app/commit/9de63f198119ad24cb63aa5cd35b0eadd7872f23))
* move tests from pre-push to pre-commit hook ([0f5cebc](https://github.com/Simplifi-ED/grafana-apm-app/commit/0f5cebcbe4aba8cb55cecb79a8048c31564200bd))
* multi-hop BFS correctness (direction-aware expansion, FQDN normalization, PromQL escaping) ([a116c4c](https://github.com/Simplifi-ED/grafana-apm-app/commit/a116c4c4b1b64a96d017cc904fcd02ef4783f2aa))
* NaN-safe numeric sort prevents broken service ordering ([589df8a](https://github.com/Simplifi-ED/grafana-apm-app/commit/589df8a337ad72d9e82e9777836c0fdf8a00c1e0))
* Node.js detection priority over Go in framework detection ([b3c13ab](https://github.com/Simplifi-ED/grafana-apm-app/commit/b3c13abeb3531e4df97c5bfe13d0870113bb485c))
* only use canvas layout for traces, logs, service-map tabs ([4bbe31a](https://github.com/Simplifi-ED/grafana-apm-app/commit/4bbe31a27e7229969b8cf87520e4eb52ca67234c))
* **ops-board:** address code review findings ([5c39dac](https://github.com/Simplifi-ED/grafana-apm-app/commit/5c39dacf2219da23d16003f0f161f1cc685e491f))
* parse namespace from double-slash ruler group paths ([#207](https://github.com/Simplifi-ED/grafana-apm-app/issues/207)) ([48d5adc](https://github.com/Simplifi-ED/grafana-apm-app/commit/48d5adc8dbfb328639e57798c7f635b0318e7d17))
* pass environment filter to service map and dependencies queries ([28aaa30](https://github.com/Simplifi-ED/grafana-apm-app/commit/28aaa3090e4e5fa92e22c3df3f7e486aaa536141))
* pass environment to MimirWebVitalsPanels for correct datasource routing ([1dbf54d](https://github.com/Simplifi-ED/grafana-apm-app/commit/1dbf54dd4b49f537a128d64d8e11308f7d34a7bb))
* patch security vulnerabilities with pnpm overrides and workflow permissions ([d1ef241](https://github.com/Simplifi-ED/grafana-apm-app/commit/d1ef241e18e81170efdc9c1f517b5c281d44d90a))
* prettier formatting and add git constraints to AGENTS.md ([a0813e6](https://github.com/Simplifi-ED/grafana-apm-app/commit/a0813e6092cd4beac2368880990b82811616f924))
* prevent RED panels from showing No Data during capability loading ([bed1380](https://github.com/Simplifi-ED/grafana-apm-app/commit/bed138008017808f296dc3b3155b177fc9317a1b))
* prevent service inventory table overflow with fixed layout ([069ce95](https://github.com/Simplifi-ED/grafana-apm-app/commit/069ce954466da726ad93e338cbf560908bc8a872))
* **queries:** close CodeQL request-forgery finding in GetAlertRules ([#102](https://github.com/Simplifi-ED/grafana-apm-app/issues/102)) ([20227bd](https://github.com/Simplifi-ED/grafana-apm-app/commit/20227bd86f147fe0c6290e69e986f027743d67af))
* reduce CWV stat panel width, give more space to Navigation Type ([b7d1a27](https://github.com/Simplifi-ED/grafana-apm-app/commit/b7d1a27d5526a444f4d93f6e7ff3ab131779aa2a))
* reduce RED panel flashing on ServiceOverview load ([2455b01](https://github.com/Simplifi-ED/grafana-apm-app/commit/2455b01d7009393fe74576ebc5684105e29f0528))
* **release:** make README links absolute + guard against relative links ([312f2ff](https://github.com/Simplifi-ED/grafana-apm-app/commit/312f2ff2aec5288d954ddbb4e7579fc05ec2a77a))
* remove $browser variable dependency from histogram Loki queries ([c6c5e3a](https://github.com/Simplifi-ED/grafana-apm-app/commit/c6c5e3a8e082a20a1e81f0b67aa5f7c7fa2499e2))
* remove broken minimap from service graph ([7389467](https://github.com/Simplifi-ED/grafana-apm-app/commit/73894675cebe099011bf4da06d54152ae271dd1f))
* remove dotted background and center service graph ([d8bc180](https://github.com/Simplifi-ED/grafana-apm-app/commit/d8bc180a40abf9ecf45fbdf820c9a4277453c7e5))
* remove empty columns from Per-Page Performance table ([3e48eea](https://github.com/Simplifi-ED/grafana-apm-app/commit/3e48eeab8a4e981874d051edd3f352ea12cdfff4))
* remove Faro badge from Type column ([88532d2](https://github.com/Simplifi-ED/grafana-apm-app/commit/88532d23c325f82e4253a75faa547491ee418766))
* remove rigid panel height constraints, let Scenes auto-size ([40bfa40](https://github.com/Simplifi-ED/grafana-apm-app/commit/40bfa40fdbee95d4235d98856df554a381031652))
* remove stale Service Map e2e test and convert README relative links ([a90ab72](https://github.com/Simplifi-ED/grafana-apm-app/commit/a90ab722462cb7e4b8369af9d96fa73e5f28fa90))
* rename alert source label to 'Prometheus Alerts' ([01c26d1](https://github.com/Simplifi-ED/grafana-apm-app/commit/01c26d1cfd06a7345d22ffa7f05c6b96d03883d6))
* render error sparkline as actual time-series chart ([872a190](https://github.com/Simplifi-ED/grafana-apm-app/commit/872a1907fbf8b2011e845c81e1139fe3ba73e5f4))
* resolve all ESLint errors ([85d5e11](https://github.com/Simplifi-ED/grafana-apm-app/commit/85d5e1127de76f70989c98faa66b1e19c661f20c))
* resolve CI warnings — remove invalid publish input, update action versions, remove deprecated config.apps ([dabf008](https://github.com/Simplifi-ED/grafana-apm-app/commit/dabf008ab299d33f1e9afdc2d8413b026a9695cc))
* resolve duplicate Authorization header in publish-report job ([94032ed](https://github.com/Simplifi-ED/grafana-apm-app/commit/94032ed34561862ecddc2ee3fd2962b32ff6506c))
* resolve duplicate uninitialized i18n instance in Scenes and deep import 404s ([b97e532](https://github.com/Simplifi-ED/grafana-apm-app/commit/b97e5327cc4781999defb3403c8c665db682dc61))
* resolve four UI bugs in endpoint table, heatmap, dependency charts, and traces ([2757d04](https://github.com/Simplifi-ED/grafana-apm-app/commit/2757d04b7ed8b83ed4c9e5db2685d86a5c277976))
* resolve infinite exception drawer reopening bug by ref-wrapping session state inside occurrences query useEffect ([2ae7dc9](https://github.com/Simplifi-ED/grafana-apm-app/commit/2ae7dc90a68ecc973c5f808794c86da16337d56d))
* resolve ops dashboard scalability and UI issues ([e4f62ea](https://github.com/Simplifi-ED/grafana-apm-app/commit/e4f62eab4d6274a5d1811ae659cf3086b24a1958))
* resolve release Go version and e2e test failures ([d13b729](https://github.com/Simplifi-ED/grafana-apm-app/commit/d13b729ea23f8cfa177573b02d8973fba472d93f))
* resolve security alerts (npm overrides + Go log/URL sanitization) ([baa0286](https://github.com/Simplifi-ED/grafana-apm-app/commit/baa028644ee71731f6f0885399719381546485ee))
* resolve TypeScript errors in ServiceInventory and ServiceOverview ([a01f711](https://github.com/Simplifi-ED/grafana-apm-app/commit/a01f7110c0876296c55fdf6ab86d3c02e13a5b0d))
* restore Dockerfile and entrypoint needed by docker-compose ([26048cb](https://github.com/Simplifi-ED/grafana-apm-app/commit/26048cbc6fe7de8219110fe227a42f9553a7eff4))
* revert display:none for Scenes-based tabs ([9f81967](https://github.com/Simplifi-ED/grafana-apm-app/commit/9f81967620599d3fecbd4bcdb69ea6ee8d2f5dd2))
* route exemplar links to environment-resolved Tempo datasource ([acb6285](https://github.com/Simplifi-ED/grafana-apm-app/commit/acb62857b6daa93d13e8b5624d2aa03ea3dade99))
* runtime pod count deduplicates by instance/pod ([fcd1b65](https://github.com/Simplifi-ED/grafana-apm-app/commit/fcd1b659353ce906e83eb0b401d34d91e78c1fe0))
* runtime utilization bars invisible due to wrong CSS variables ([a0e5e4b](https://github.com/Simplifi-ED/grafana-apm-app/commit/a0e5e4b93cd163af3717c19e35421695fabe6a95))
* sanitize corrupted environment param on all pages ([228197a](https://github.com/Simplifi-ED/grafana-apm-app/commit/228197a777fe2ce3a2390c2c2f9b514edfebc44e))
* sanitize corrupted environment params from browser history ([4c984ab](https://github.com/Simplifi-ED/grafana-apm-app/commit/4c984ab80cbf499e6f19c4f1819231a061e52cde))
* scope Server tab operations to selected environment ([2892067](https://github.com/Simplifi-ED/grafana-apm-app/commit/28920674775d6d516df5cb5dfc379615884e5410))
* scope service graph queries by service for per-service topology ([1666fb1](https://github.com/Simplifi-ED/grafana-apm-app/commit/1666fb1e07d01e59311914eb919875396003bd91))
* **scorecard:** mark browser telemetry not-applicable for pure backends ([#142](https://github.com/Simplifi-ED/grafana-apm-app/issues/142)) ([60db992](https://github.com/Simplifi-ED/grafana-apm-app/commit/60db992580c621ce1a3160f743e9c08efdf338c5))
* search http.route in addition to span name for trace lookups ([401c5d5](https://github.com/Simplifi-ED/grafana-apm-app/commit/401c5d5eb8231ce408832093ac72f8e7073fafb9))
* security and performance hardening ([5e4bec1](https://github.com/Simplifi-ED/grafana-apm-app/commit/5e4bec1c233996d4ca4021181d2d39160b87ee7d))
* service filter, column alignment, and name truncation on team page ([2442ef5](https://github.com/Simplifi-ED/grafana-apm-app/commit/2442ef5e3bf00652a428e89b06102e79d7221a9d))
* service graph polish — Grafana icons, arrows, neutral borders, tighter layout ([6c564d1](https://github.com/Simplifi-ED/grafana-apm-app/commit/6c564d10d436818b9581361ff65c0fd33cfe2d8a))
* service inventory sort instability on sparkline arrival ([9c15579](https://github.com/Simplifi-ED/grafana-apm-app/commit/9c15579e6e4504cebc76a873ca95480225b28b77))
* service list duplicate rows and unreliable filtering ([dbbfd58](https://github.com/Simplifi-ED/grafana-apm-app/commit/dbbfd58826bd2fcf455d2f4703ab10111a7dfada))
* show errors on failure, brief summary on success ([2c82bd5](https://github.com/Simplifi-ED/grafana-apm-app/commit/2c82bd54aab5c48a95062e85e72a33a4efc48445))
* show low-traffic services across selected time range ([#193](https://github.com/Simplifi-ED/grafana-apm-app/issues/193)) ([64d1735](https://github.com/Simplifi-ED/grafana-apm-app/commit/64d17355a5041a00950e3a0d2e09ada8d298a8e9))
* show only message in logs instead of raw JSON ([3d597d5](https://github.com/Simplifi-ED/grafana-apm-app/commit/3d597d5378dfba237e882609f987ea0a8575a73e))
* show RED metrics and sparklines for non-server services ([4ad843d](https://github.com/Simplifi-ED/grafana-apm-app/commit/4ad843d826b1e2c95b6067f2813f44d15c080cad))
* show topology graph regardless of environment filter ([7aac4a2](https://github.com/Simplifi-ED/grafana-apm-app/commit/7aac4a23a1510960b7303d753f32c3c7379b1b8c))
* shrink Type column to fit content width ([96c8f4e](https://github.com/Simplifi-ED/grafana-apm-app/commit/96c8f4ee1b5930f064700ee65bd38bf3b5a7f8bd))
* simplify CWV Rating panel title ([2838d26](https://github.com/Simplifi-ED/grafana-apm-app/commit/2838d266b1b6a68215ea4700906761b38c852c64))
* skip infrastructure type for nodes that are also clients ([c99a838](https://github.com/Simplifi-ED/grafana-apm-app/commit/c99a8380339975c650df4e1fb17bdfcf471d7184))
* skip namespace filter for per-service topology graph ([f5ab5c8](https://github.com/Simplifi-ED/grafana-apm-app/commit/f5ab5c83cf97f099c2349356e3d1c38f5ecbc137))
* sort direction applied only to primary field, not tiebreaker ([b4c801e](https://github.com/Simplifi-ED/grafana-apm-app/commit/b4c801e45c1e466ef59309eb361d17759314c5a6))
* sort environment groups prod-first in namespace view ([4dc97f0](https://github.com/Simplifi-ED/grafana-apm-app/commit/4dc97f0e8c995beab2489f806cb94c536e8bbaf9))
* split errors row layout and fix exception-to-logs linking ([6f1a777](https://github.com/Simplifi-ED/grafana-apm-app/commit/6f1a77734432d7da979add61b01ba04270263aed))
* **status-board:** fix grid showing 1 column ([e36d2ff](https://github.com/Simplifi-ED/grafana-apm-app/commit/e36d2fffd7e9a855cf6a6815c8a359d6da95d488))
* **status-board:** show service name on medium cards ([73cae27](https://github.com/Simplifi-ED/grafana-apm-app/commit/73cae2755e7be33b692fcc61f1c50018bf223a81))
* style Faro badge as pill with spacing from framework badge ([ab8429e](https://github.com/Simplifi-ED/grafana-apm-app/commit/ab8429e2c6697bc98791c9b7f793e4a4ea7b0974))
* **test:** dedupe react-router so grafana-ui links see the test Router ([#169](https://github.com/Simplifi-ED/grafana-apm-app/issues/169)) ([6c8eb0f](https://github.com/Simplifi-ED/grafana-apm-app/commit/6c8eb0fc9512d91aee07973d6be5ade294301a81))
* traces and logs tabs not filling full page height ([8ffde78](https://github.com/Simplifi-ED/grafana-apm-app/commit/8ffde78ccb460e5e92bf44a76370e226e3d7abab))
* type column alignment using inner flex wrapper instead of flex td ([949a4f8](https://github.com/Simplifi-ED/grafana-apm-app/commit/949a4f84fae55971e580e2a4b2fc3703620eebca))
* **ui:** align environment override rows on config page ([fe24f4a](https://github.com/Simplifi-ED/grafana-apm-app/commit/fe24f4a577c79a92ab0c544bc3a2b59b08e0634c))
* **ui:** reduce heatmap height and improve topology fitView ([bb29acf](https://github.com/Simplifi-ED/grafana-apm-app/commit/bb29acfb6e778f6cd7c051a08f78e21fcaad6ef2))
* use '_' placeholder for empty namespace in exception link URL ([72e5e80](https://github.com/Simplifi-ED/grafana-apm-app/commit/72e5e80c5059187c12ed29628deeeb4a32569673))
* use breadcrumb test IDs in e2e navigation tests ([067ec6c](https://github.com/Simplifi-ED/grafana-apm-app/commit/067ec6c55133cc5b32f809c9e6f85a93daf04274))
* use correct environment keys in provisioning config ([cd825ce](https://github.com/Simplifi-ED/grafana-apm-app/commit/cd825ce50285a9bc02323978e47bdca39bcc9bcf))
* use CSS locator with .first() for carousel dismiss ([cf1856a](https://github.com/Simplifi-ED/grafana-apm-app/commit/cf1856a89a1ac9c2175b998b10f5d602294241bb))
* use detected durationMetric in frontend instead of constructing it ([155995f](https://github.com/Simplifi-ED/grafana-apm-app/commit/155995f5ace54735ad2b0cac723ab08cd1bf6fb4))
* use FCP histogram as pageload proxy for accurate count ([9a938e8](https://github.com/Simplifi-ED/grafana-apm-app/commit/9a938e8f3f39e6459d91146645c367805654800e))
* use full time range for histogram stat panels ([fb74e62](https://github.com/Simplifi-ED/grafana-apm-app/commit/fb74e624875fbaddd04dfd31c88573c5428a9086))
* use golangci-lint v2 config schema ([0b93431](https://github.com/Simplifi-ED/grafana-apm-app/commit/0b934316821634dfc8cafaa6c85e72b80c52b4c2))
* use Grafana Tooltip component for bullet chart help icons ([53d07ab](https://github.com/Simplifi-ED/grafana-apm-app/commit/53d07ab4fd784ae75ba09904fc4f4b47081c5a1c))
* use localhost for internal API calls and propagate auth headers ([9df0254](https://github.com/Simplifi-ED/grafana-apm-app/commit/9df0254bd288218083af828eedf96750818fe2d7))
* use object format for plugin.json extensions (fixes validator) ([5b32854](https://github.com/Simplifi-ED/grafana-apm-app/commit/5b32854b7527e98e5bd0d1c5d0629c34f928a07e))
* use sandbox-safe wrapper in release script ([39e0fd6](https://github.com/Simplifi-ED/grafana-apm-app/commit/39e0fd607b6b964281ed98b95a67efdf1f507190))
* use URL assertion for service inventory e2e test ([ffd79dc](https://github.com/Simplifi-ED/grafana-apm-app/commit/ffd79dc73f9d01769d93ffe52f474666e5c1b4d3))
* use useSearchParams directly to perform atomic parameter updates on close and prevent loop ([46cdfca](https://github.com/Simplifi-ED/grafana-apm-app/commit/46cdfcab0d28c52f3178e6285ec2c7f48fb98f61))
* use weighted mean for Loki vital queries to match Scene panels ([cc4a57c](https://github.com/Simplifi-ED/grafana-apm-app/commit/cc4a57c9050932f4ef14fef713d890678a48b6b7))
* **v0.20.1:** triage IAM permissions + Logs tab LogQL escaping ([a8b4324](https://github.com/Simplifi-ED/grafana-apm-app/commit/a8b43242417fcfbe3c59ec7e115e1f72f1caf5c0))
* widen hasLokiFaroData detection window from 1h to 6h ([edcdb81](https://github.com/Simplifi-ED/grafana-apm-app/commit/edcdb81882787b54cb1ef05c1956cdd7464e19c5))
* widen histogram detection window from 1h to 6h ([46ac307](https://github.com/Simplifi-ED/grafana-apm-app/commit/46ac307e3f3ceb04ab10c6f52672da7d2b17cc82))


### Performance Improvements

* major perceived performance improvements ([5475d4a](https://github.com/Simplifi-ED/grafana-apm-app/commit/5475d4a79040f26b9a061b89ff7ed46b1a25e9e4))
* reduce service inventory query load ([#202](https://github.com/Simplifi-ED/grafana-apm-app/issues/202)) ([3118680](https://github.com/Simplifi-ED/grafana-apm-app/commit/3118680c44081c18300c119217d0d8f673ecd03d))


### Documentation

* add 0.10.0 changelog entry ([7ab1644](https://github.com/Simplifi-ED/grafana-apm-app/commit/7ab16446b4f5112831cb1b212e914ef2867f3e04))
* add 0.10.1 changelog entry ([0219d9a](https://github.com/Simplifi-ED/grafana-apm-app/commit/0219d9a3a21781c99109f40af2b9cb4965b40c0b))
* add 0.6.0 changelog ([2ff83f2](https://github.com/Simplifi-ED/grafana-apm-app/commit/2ff83f2895d2284c5241a3f6f16bab3c9d2fb651))
* add 0.6.1 changelog ([bda9975](https://github.com/Simplifi-ED/grafana-apm-app/commit/bda9975eacc011ab7596a6ccb81f7b4be94fbb45))
* add 0.6.2 changelog ([31aa4b6](https://github.com/Simplifi-ED/grafana-apm-app/commit/31aa4b6cb60b2459924522af7e9da91e0e120246))
* add agent constraints to AGENTS.md ([d8d1ed1](https://github.com/Simplifi-ED/grafana-apm-app/commit/d8d1ed1d5540b1f5ec420142d8a53c7495ca9e5c))
* add AGENTS.md with repo structure and conventions ([29622b8](https://github.com/Simplifi-ED/grafana-apm-app/commit/29622b8053654f821eb0e72a6457db8b60b78cf8))
* add changelog for upcoming release ([d89727a](https://github.com/Simplifi-ED/grafana-apm-app/commit/d89727aa3ed7cfd089857f437060796c660d6271))
* add design philosophy with UX principles and observability strategy ([3a26533](https://github.com/Simplifi-ED/grafana-apm-app/commit/3a26533f030006df5c01c82b725f839c4781738f))
* add i18n gotchas and backend compile rules to AGENTS.md ([a75f9ff](https://github.com/Simplifi-ED/grafana-apm-app/commit/a75f9ff812a2574ae2a6166e40c61fe75a8dcada))
* add metrics, labels, and configuration reference ([ebac6ff](https://github.com/Simplifi-ED/grafana-apm-app/commit/ebac6ffe0065d5f996dbd090dbb501683b1cf758))
* add release task to AGENTS.md commands ([a10ff88](https://github.com/Simplifi-ED/grafana-apm-app/commit/a10ff8830674f5a8adecfa124a8bb51de2a8f9e2))
* add server endpoints and frontend screenshots to README and plugin.json ([4f12190](https://github.com/Simplifi-ED/grafana-apm-app/commit/4f121907e666223b29127f89a922023b8e22fa20))
* add service discovery fix to changelog ([b00be53](https://github.com/Simplifi-ED/grafana-apm-app/commit/b00be53ac914fcce949ad8b0dad9990f2bfebcaa))
* add unreleased changelog for v0.4.4 refactoring ([7b41dbb](https://github.com/Simplifi-ED/grafana-apm-app/commit/7b41dbbceec54945be4f3414ede79c0ae1548c13))
* add unreleased changelog for v0.5.1 fixes ([c029c84](https://github.com/Simplifi-ED/grafana-apm-app/commit/c029c84a9ebe487ec92597685280a7d4d517872e))
* add v0.12.0 release notes ([850206f](https://github.com/Simplifi-ED/grafana-apm-app/commit/850206fadbb97036f451226512f60742e19bc659))
* add v0.12.1 release notes ([bb2a214](https://github.com/Simplifi-ED/grafana-apm-app/commit/bb2a214127e061d5fcdd329a8ae3feb1b0069ff7))
* add v0.7.0 changelog entry for Mimir-first refactor ([d57ffe7](https://github.com/Simplifi-ED/grafana-apm-app/commit/d57ffe74b3d661d4371e14af6e3b4020a7e49469))
* add v0.7.1 changelog entry ([f0b0750](https://github.com/Simplifi-ED/grafana-apm-app/commit/f0b0750e84f1a15f3499186244d76ab15001e607))
* add visualization guide with component decision table ([234c889](https://github.com/Simplifi-ED/grafana-apm-app/commit/234c889e253c58437a8aa84532ace093fa30e3bf))
* **adr:** backend stack traces via OpenTelemetry, not log-pipeline merging ([#129](https://github.com/Simplifi-ED/grafana-apm-app/issues/129)) ([e1c20dd](https://github.com/Simplifi-ED/grafana-apm-app/commit/e1c20ddc1bd0c80272f66005caee883a2deb980f))
* agent output and issue drafts ([28f9c20](https://github.com/Simplifi-ED/grafana-apm-app/commit/28f9c20b55b6234ef30bdea0dae90dd4b514185c))
* **changelog:** add unreleased release notes ([3849eaf](https://github.com/Simplifi-ED/grafana-apm-app/commit/3849eafe3ee1e6cbff13dc91e6f65bc89f71ff95))
* comprehensive metrics pipeline and operator configuration guide ([6e830a9](https://github.com/Simplifi-ED/grafana-apm-app/commit/6e830a942de587eb9153bd58dca6fc1144c8482d))
* document all metrics, traces, and log data with origins ([76ded08](https://github.com/Simplifi-ED/grafana-apm-app/commit/76ded08b3c0c8c0a2b136d6adffc2f9127dea839))
* enrich design philosophy with additional research sources ([7dad5ac](https://github.com/Simplifi-ED/grafana-apm-app/commit/7dad5ace05c4cc91a76e19326c999f64c7593f9f))
* expand Configuration section with per-env overrides and auto-detect ([78d8432](https://github.com/Simplifi-ED/grafana-apm-app/commit/78d843227d66082d6a4d0f7516b7d874250e5ff9))
* mark legacy frontend panel components as deprecated ([1b99fc9](https://github.com/Simplifi-ED/grafana-apm-app/commit/1b99fc99407ca12f95974e7eec135dc677ef77da))
* restructure documentation ([9250cc1](https://github.com/Simplifi-ED/grafana-apm-app/commit/9250cc18aa41c6370de99fd57c698d6c57750fa2))
* rewrite panel descriptions in plain language ([b60992e](https://github.com/Simplifi-ED/grafana-apm-app/commit/b60992e9582693a542cbe986c712ac5792e4bb85))
* sharpen README pitch — problem→solution→features ([2699ddd](https://github.com/Simplifi-ED/grafana-apm-app/commit/2699dddbc0607381553e39277c123fa7c69fbfaa))
* sync ClickStack parity matrix after enablement smoke ([2a5c439](https://github.com/Simplifi-ED/grafana-apm-app/commit/2a5c439ebbbebaa0c07fb201d9582be1a67bc98e))
* update 0.5.3 changelog with full release notes ([3d71ef4](https://github.com/Simplifi-ED/grafana-apm-app/commit/3d71ef4eeadcdf76eab55024eb6aa6f02bee253f))
* update changelog and readme for namespace overview and fixes ([0ec335b](https://github.com/Simplifi-ED/grafana-apm-app/commit/0ec335bb828e9e26731de1d6ab9b8600d6291698))
* update changelog and readme for session changes ([21424dc](https://github.com/Simplifi-ED/grafana-apm-app/commit/21424dc3edf9ed27f30545f00a131d694505c21a))
* update changelog and readme for v0.5.0 release ([357cd06](https://github.com/Simplifi-ED/grafana-apm-app/commit/357cd06cbe188819b5b6b2b67811e390036f1b79))
* update CHANGELOG for v0.1.0 release ([bc265cc](https://github.com/Simplifi-ED/grafana-apm-app/commit/bc265cc02cc43cb4edc29adbae86b1f907203dbd))
* update ClickStack migration feature matrix and enhance SQL queries ([695f98d](https://github.com/Simplifi-ED/grafana-apm-app/commit/695f98d21782ba23fc7a172e4e69f22b6a4fde1e))
* update ClickStack migration feature matrix to reflect recent UI smoke tests ([d69ff39](https://github.com/Simplifi-ED/grafana-apm-app/commit/d69ff3939979fed19d8e28468908a6427d22e6fd))
* update README, CONTRIBUTING, AGENTS, and CHANGELOG ([eae97e0](https://github.com/Simplifi-ED/grafana-apm-app/commit/eae97e01ced7a58bc0cba61ed52c89d2150bae2e))
* update screenshots for README.md ([37223d7](https://github.com/Simplifi-ED/grafana-apm-app/commit/37223d77b710bdfc8c868ebd5b124850d716ef77))
* update stale references and add 0.5.3 changelog ([bfd2446](https://github.com/Simplifi-ED/grafana-apm-app/commit/bfd2446bdc8ebf9237926417b494458ea560eed7))
* update v0.7.0 changelog with kind filter and link fixes ([2a16800](https://github.com/Simplifi-ED/grafana-apm-app/commit/2a16800a1d1bd0e77dae61d7154757d7e4e7d796))

## [0.25.3](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.25.2...nais-apm-app-v0.25.3) (2026-10-15)


### Performance Improvements

* reduce service inventory query load ([#202](https://github.com/nais/grafana-apm-app/issues/202)) ([3118680](https://github.com/nais/grafana-apm-app/commit/3118680c44081c18300c119217d0d8f673ecd03d))

## [0.25.2](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.25.1...nais-apm-app-v0.25.2) (2026-10-14)


### Bug Fixes

* clarify release commit titles ([#200](https://github.com/nais/grafana-apm-app/issues/200)) ([509c644](https://github.com/nais/grafana-apm-app/commit/509c644b435b66066d8a96e02d335f10199cc6d8))

## [0.25.1](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.25.0...nais-apm-app-v0.25.1) (2026-10-12)


### Bug Fixes

* filter logs when environments share Loki ([#195](https://github.com/nais/grafana-apm-app/issues/195)) ([fde59fb](https://github.com/nais/grafana-apm-app/commit/fde59fbc6ade98b35a49a0dc7d8fd653bcb0eba5))

## [0.25.0](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.24.0...nais-apm-app-v0.25.0) (2026-09-25)


### Features

* **alerts:** new issues → Slack via a copy-pasteable PrometheusRule ([#177](https://github.com/nais/grafana-apm-app/issues/177)) ([257c88c](https://github.com/nais/grafana-apm-app/commit/257c88cfaa7e34b8d447a0e99d9b1ed134544e02))
* **scorecard:** make every failing readiness check one click to fix ([#176](https://github.com/nais/grafana-apm-app/issues/176)) ([7abe3ad](https://github.com/nais/grafana-apm-app/commit/7abe3ad4db8cd0540f2f8cf97b0c09b6e0cad11d))


### Bug Fixes

* **build:** drop tsconfig baseUrl ahead of the TypeScript 7 bump ([#179](https://github.com/nais/grafana-apm-app/issues/179)) ([a6399fc](https://github.com/nais/grafana-apm-app/commit/a6399fc045519cb992cd6ceb751c01c5c8ebffc5))
* **deps:** pin transitive dependencies flagged by Dependabot ([#178](https://github.com/nais/grafana-apm-app/issues/178)) ([8cfbc1f](https://github.com/nais/grafana-apm-app/commit/8cfbc1f34881815ca30423c831160215c58639aa))
* show low-traffic services across selected time range ([#193](https://github.com/nais/grafana-apm-app/issues/193)) ([64d1735](https://github.com/nais/grafana-apm-app/commit/64d17355a5041a00950e3a0d2e09ada8d298a8e9))

## [0.24.0](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.23.1...nais-apm-app-v0.24.0) (2026-09-05)


### Features

* **naissync:** prune stale deploy annotations ([#173](https://github.com/nais/grafana-apm-app/issues/173)) ([dba7c66](https://github.com/nais/grafana-apm-app/commit/dba7c66c526daed24e5d4ea2697eb1c0890a5e47)), closes [#128](https://github.com/nais/grafana-apm-app/issues/128)


### Bug Fixes

* **issues:** collapse stdout copy of an OTLP-exported exception ([#171](https://github.com/nais/grafana-apm-app/issues/171)) ([1c3bab0](https://github.com/nais/grafana-apm-app/commit/1c3bab0068b39ed8056368def5619ad93ca876eb))

## [0.23.1](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.23.0...nais-apm-app-v0.23.1) (2026-09-05)


### Bug Fixes

* **issues:** anchor the server-occurrence scan so drawer details match the list ([#157](https://github.com/nais/grafana-apm-app/issues/157)) ([0bade9b](https://github.com/nais/grafana-apm-app/commit/0bade9b7cd886489955439a32f8c820a080a0b99))
* **test:** dedupe react-router so grafana-ui links see the test Router ([#169](https://github.com/nais/grafana-apm-app/issues/169)) ([6c8eb0f](https://github.com/nais/grafana-apm-app/commit/6c8eb0fc9512d91aee07973d6be5ade294301a81))

## [0.23.0](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.22.1...nais-apm-app-v0.23.0) (2026-08-03)


### Features

* **seed:** deterministic synthetic Faro telemetry generator ([#135](https://github.com/nais/grafana-apm-app/issues/135)) ([dcf3c97](https://github.com/nais/grafana-apm-app/commit/dcf3c97c6472ccc62db13929b4395bdae5a98961))


### Bug Fixes

* **build:** migrate TypeScript config for version 7 ([#155](https://github.com/nais/grafana-apm-app/issues/155)) ([fa27c8d](https://github.com/nais/grafana-apm-app/commit/fa27c8dfdd98695571f966f99186f81e7b1304ff))
* **grafana:** support Grafana 13 local development ([#152](https://github.com/nais/grafana-apm-app/issues/152)) ([dbd7dc3](https://github.com/nais/grafana-apm-app/commit/dbd7dc32df12074d9f1e5f485c544fbfaaac3a93))
* **scorecard:** mark browser telemetry not-applicable for pure backends ([#142](https://github.com/nais/grafana-apm-app/issues/142)) ([60db992](https://github.com/nais/grafana-apm-app/commit/60db992580c621ce1a3160f743e9c08efdf338c5))

## [0.22.1](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.22.0...nais-apm-app-v0.22.1) (2026-07-07)


### Documentation

* **adr:** backend stack traces via OpenTelemetry, not log-pipeline merging ([#129](https://github.com/nais/grafana-apm-app/issues/129)) ([e1c20dd](https://github.com/nais/grafana-apm-app/commit/e1c20ddc1bd0c80272f66005caee883a2deb980f))

## [0.22.0](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.21.0...nais-apm-app-v0.22.0) (2026-07-06)


### Features

* **database:** Phase 2 — Top queries (normalized, Tempo-sourced) ([#119](https://github.com/nais/grafana-apm-app/issues/119)) ([#124](https://github.com/nais/grafana-apm-app/issues/124)) ([a6c427d](https://github.com/nais/grafana-apm-app/commit/a6c427d2d10e53802ab99fef75da5688d4b248d5))
* **issues:** richer context for backend log occurrences ([#118](https://github.com/nais/grafana-apm-app/issues/118)) ([b77673b](https://github.com/nais/grafana-apm-app/commit/b77673bf6c8c4e0438fc7fd81b13e731ac0f100d))

## [0.21.0](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.20.2...nais-apm-app-v0.21.0) (2026-07-05)


### Features

* **alerting:** harden defaults= contract ([#77](https://github.com/nais/grafana-apm-app/issues/77)) + surface firing state ([#33](https://github.com/nais/grafana-apm-app/issues/33) Phase 1) ([#104](https://github.com/nais/grafana-apm-app/issues/104)) ([85b6a88](https://github.com/nais/grafana-apm-app/commit/85b6a88589913a8e617f12c051c9a8bd1b0db95d))
* **alerts:** read-only firing-alert detail drawer ([#32](https://github.com/nais/grafana-apm-app/issues/32)) ([#110](https://github.com/nais/grafana-apm-app/issues/110)) ([2996d1e](https://github.com/nais/grafana-apm-app/commit/2996d1eaaa734606a1ad6544d1840aa047d7588c))
* **frontend:** detect minified stacks + source-map doctor ([#60](https://github.com/nais/grafana-apm-app/issues/60)) ([#103](https://github.com/nais/grafana-apm-app/issues/103)) ([c402ffa](https://github.com/nais/grafana-apm-app/commit/c402ffa248f496f1a4cbcde688352cc14c2f95cc))
* **issues:** server-issue drawer parity — open the exception drawer for backend errors ([#84](https://github.com/nais/grafana-apm-app/issues/84) Phase 1) ([#93](https://github.com/nais/grafana-apm-app/issues/93)) ([458950a](https://github.com/nais/grafana-apm-app/commit/458950a34d0d3e5512781fb17359ef3d244eb0f2))


### Bug Fixes

* **queries:** close CodeQL request-forgery finding in GetAlertRules ([#102](https://github.com/nais/grafana-apm-app/issues/102)) ([20227bd](https://github.com/nais/grafana-apm-app/commit/20227bd86f147fe0c6290e69e986f027743d67af))

## [0.20.2](https://github.com/nais/grafana-apm-app/compare/nais-apm-app-v0.20.1...nais-apm-app-v0.20.2) (2026-07-05)


### Bug Fixes

* **config:** allow clearing the service account token + de-bloat the page ([#92](https://github.com/nais/grafana-apm-app/issues/92)) ([b6afbc2](https://github.com/nais/grafana-apm-app/commit/b6afbc2fd57a5cde9824a3a96b1e36e4e631aba1))

## 0.20.1 — "Legend" patch (2026-07-05)

### Fixes

- **Triage state failed with 502 in production** — the plugin's `iam.permissions` (plugin.json) declared only `datasources:query`, so Grafana's auto-provisioned managed service account had no access to the annotations API that triage reads/writes. Added `annotations:read`/`create`/`write`. Also unblocks deploy-marker annotations, which use the same service account.
- **Logs tab failed to render on issue deep-links** — a search term containing a regex metacharacter (e.g. `[` from an Issues-table "view logs" link) was regex-escaped but not string-escaped, producing an invalid `\[` LogQL string escape that Loki rejected. `LogsTab` now applies `escapeQueryString` around the regex, matching the Traces tab.

## 0.20.0 — "Legend" (2026-07-05)

### Features

- **Fingerprint-grouped exceptions** — Top Exceptions now groups by a stable, versioned fingerprint computed in the backend (exception type + normalized message) instead of the raw per-message Alloy hash: one logical error is one row even when messages carry UUIDs, URLs, or timestamps, with a "merged ×N" badge showing how many raw hashes combined. The Exception Drawer opens via a shareable `issueId` and aggregates occurrences across all member hashes; legacy `exceptionHash` links keep working. (#62)
- **Frontend tab follows the page time range** — The Frontend tab's panels were pinned to a hardcoded last-1h window; they now honor the shared `from`/`to` URL range like every other tab.
- **Top Exceptions promoted to a full-width row** — The issues table now sits directly under the health stats with the full page width, each row showing a share-of-total bar; Exception Types moved down beside the browser panels. Long issue lists paginate 10 rows at a time, and the backend caps the payload at the top 100 groups.
- **Deploy markers on charts** — RED and Frontend panels render deployment annotations (org annotations tagged `nais-apm:deploy`) as markers with version and workflow link in the tooltip; a reusable GitHub Action for writing them ships under `infra/actions`. (#64)
- **One-click alert rules** — "Create alert" in the Exception Drawer and "Alert on error rate" in the service header open Grafana's alert editor pre-filled with the right query (detected metric names, fingerprint-scoped exception spikes with a deep link back to the drawer, web-vitals degradation). Rules belong to the team, not the plugin. (#65)
- **Grafana Drilldown integration** — "Open in Logs Drilldown" from the Logs tab, and an "Open in Nais APM" link on Grafana's Explore toolbar that recognizes `service_name` in the current query.
- **Versions panel** — per-release sessions, adoption share, error-free rate, and exception counts under Top Exceptions, enriched with deploy timestamps from deploy annotations ("did this start with today's release?"). (#64)
- **Web-vitals attribution** — three new panels show the slowest LCP elements, INP interaction targets, and CLS layout-shift sources from the attribution data Faro v2 already ships.
- **Readable stack traces for console-captured exceptions** — The Exception Drawer now collapses runs of Faro SDK, vendored-dependency, and browser-native frames behind an expandable toggle and highlights the first in-app frame as the likely error origin. Exceptions captured via `console.error` get an explanatory badge: their stack shows the console call site, not where the error was thrown. Works retroactively on existing Loki data. (#66)
- **Grafana-managed alerts on the namespace page** — Namespace alert surfacing now merges Grafana unified-alerting rules (with a "Grafana alert" badge) alongside Mimir ruler rules, fetched in parallel and deduplicated by name. (#65)
- **Fuzzy service search** — The service inventory search matches partial, misspelled, and out-of-order tokens (e.g. `paymnt svc` finds `nais-payment-service`), ranked by relevance with exact prefix matches first. (#61)

- **Issue triage** — resolve, ignore, reopen, and assign exceptions, shared across all users and Grafana replicas via an append-only event log in org annotations (zero new infrastructure; the log is the audit history). The Top Exceptions list gains an Unresolved/All/Resolved/Ignored filter (default hides handled issues), per-user "mute for me", and a Regressed badge that bubbles resolved-but-recurring issues to the top after a newer deploy. The drawer gets a triage bar with assignee. (#57)
- **Automatic deploy annotations** — configure `naisApiUrl` + `naisApiToken` and the backend mirrors successful nais deployments as deploy markers, idempotently — no per-team CI step needed. (#64)
- **Unified Issues: backend errors join the list** — the Top Exceptions panel now also groups exceptions from the service's own backend logs (OTLP structured-metadata and JSON logstash shapes) through the same fingerprint pipeline, with browser/server source badges, a source filter, pod-impact counts, pre-filtered Logs deep links, and example-trace links via event-scope TraceQL (Tempo ≥ 2.6). Triage works identically for server issues. (#63)
- **Session replay & crash snapshots (opt-in)** — apps using `@nais/apm` can enable error-triggered session replay (`sessionReplay`) or lightweight masked DOM snapshots on crash (`screenshotOnError`); both default off, masked at capture time with a non-overridable privacy floor. The Exception Drawer plays recordings (seeked to just before the error) or shows the snapshot via a lazy-loaded player that non-replay users never download. (#58, #67)
- **Sessions explorer** — search sessions by user or id, see per-session error counts and metadata, and jump straight into the session's logs. (roadmap M5)
- **"New exceptions" alert template** — a fourth one-click alert rule fires per exception hash first seen in the last 30 minutes (7-day lookback, honest about its approximations in the alert text). (#65)

- **Dedicated Issues tab** — the unified triage surface (browser and server exceptions, releases, sessions) is now its own top-level tab; the Frontend tab is reordered vitals-first so each surface has a single clear job. (#69)
- **Database tab** — per-service query analytics and connection-pool health derived from database span metrics, verified across MongoDB, Oracle, and PostgreSQL apps; the empty states double as interim instrumentation docs. (#14)
- **Service health header** — the Overview page leads with an at-a-glance health signal and baseline deltas versus the previous period (a pragmatic stand-in for anomaly detection), in deploy-marker context. (#35)
- **Log patterns** — the Logs tab gains a Patterns view that clusters high-volume log lines via Loki's pattern ingester, plus a "new error patterns" card, to show the shape of an incident faster than scrolling raw logs.
- **Trace analytics** — TraceQL-metrics breakdowns group latency and error rate by span attribute ("which tag explains the p99") on Tempo instances that expose the metrics API.
- **Faceted issue search** — narrow the Issues list by app version, browser, or page; facet values are discovered from the data and offered as dropdowns, with a triage-status filter alongside. (M6)
- **Global time picker** — a single page-header time picker drives every tab and fleet page through the shared `from`/`to` URL params, replacing the per-tab pickers.
- **User feedback** — apps using `@nais/apm`'s feedback API stream user feedback to Loki; matching entries are joined to issues by fingerprint and shown inline in the Exception Drawer.
- **Custom application metrics** — zero-config auto-discovery surfaces app-defined metrics and Faro custom measurements without denylist maintenance. (#68)
- **Auto-refresh on inventory pages** — the non-Scene pages (service inventory, jobs, service map) now refresh on an interval like the Scene-based tabs. (#30)
- **SLO error budgets** — multi-window burn-rate alert-rule templates plus an error-budget panel on the Overview tab, computed from RED metrics — a self-hosted answer to cloud-only SLO tooling.
- **Profiling tab** — a capability-gated Pyroscope tab with span-to-profile links, shown only when a Pyroscope datasource is detected. (M7)
- **Service scorecards** — an observability-readiness score per service, enriched with ownership, runbook, and repository links from the nais Console API. (#73)
- **Jobs monitoring** — a cron / Naisjob inventory built from kube-state-metrics: schedules, last and next run, failure streaks, and one-click logs. (#74)
- **Global service map** — the fleet-wide service map returns with namespace clustering, replacing the all-services graph removed in 0.5.0. (#22)
- **In-plugin documentation links** — contextual links to the user docs and the `@nais/apm` SDK throughout the UI.

### Fixes

- **Session counts zeroed out on wide time ranges** — the distinct-sessions query exceeds Loki's series limit for chatty apps; it now retries over a narrowing recent window (labeled in the column header, e.g. "Sessions (last 5m)") and shows a dash instead of a misleading 0 when even that fails. Exception queries now use Grafana's `/api/ds/query` API instead of the deprecated datasource proxy.
- **Column sorting was dead while searching** — fuzzy-search relevance ranking silently overrode explicit column sorts; relevance is now only the default order during a search.

- **Infrequent dependencies missing from the service map** — The service-graph rate window now spans the full selected time range (floor 5m, cap 24h) instead of capping at 1h, so hourly batch jobs and cron-driven dependencies stay visible on wide dashboard ranges. (#36)
- **Status boards re-queried a frozen time window** — Auto-refresh on the Ops Status Board and namespace Status Board now re-resolves relative time ranges (`now-1h`) each tick instead of re-querying the window resolved at first page render forever.

- **Server issues matched real backend log shapes** — the unified Issues server side now recognizes the OTLP/semconv, JSON-body (logback/logstash, pino, slog), and unstructured plain-text log shapes seen in production, so backend exceptions actually appear instead of being silently missed; error-level lines with no titleable content surface as a single "Unparsed error logs" group rather than vanishing. (#63)
- **Connection-pool runtime metrics** — pool utilization now keys on the OTel `pool_name` label, fixing missing or merged pool rows on the Runtime tab.
- **Duplicated error panels removed** — the Overview and Frontend tabs no longer render the same error panel twice (IA review P4/P5).
- **Logs tab severity filter dropped plain-text logs** — filtering now matches Loki's `detected_level` structured metadata instead of parsing the log body as JSON, so non-JSON (plain-text, logfmt) logs are no longer silently excluded from the severity filter and volume panel.
- **Web Vitals enrichment now reports p75** (per the Core Web Vitals definition and matching the histogram panels) instead of the mean.

### Performance

- **Request coalescing and wider caching** — Concurrent cache misses for the same key (wallboards polling in lockstep) now share a single backend query fan-out; all read endpoints (health, operations, endpoints, runtime, dependencies, connected, graphql) are cached; capability probes are coalesced so cache expiry can't stampede detection queries; `/services` sparkline resolution is clamped server-side to ≤50 points per series and the inventory fetches sparklines for the visible page only; namespace service-graph queries are scoped to the namespace's services instead of fetching all fleet edges; label-values lookups are bounded to the request time range; the Exception Drawer's Loki queries use line prefilters and explicit time bounds. (#69)
- **Fewer redundant fetches** — Dependency detail no longer fetches fleet-wide sparkline series for its namespace lookup, and label-override settings no longer produce a fresh object per render (which cascaded into scene rebuilds and drawer refetch loops).

### Documentation

- **Development roadmap** — the milestone/track sequencing across the plugin, `@nais/apm` SDK, and platform tracks now lives in GitHub Milestones and roadmap tracking issue #91.
- **Stable URL contract** — `docs/url-contract.md` freezes the Exception Drawer deep-link parameters as a stable API for alert annotations and shared links, and now also documents the Issues/Database/Profiling tabs, the Jobs and Service Map fleet routes, the issue facet params, and the global header time range.
- **Resource API reference** — a documented, versioned resource API with a drift-proof OpenAPI spec. (#75)

## 0.13.4 (2026-07-01)

### Fixes

- **Exception Drawer Close Loop** — Solved a React state update race condition where closing the Exception Drawer would update parameters sequentially. Both the `exceptionHash` and `exceptionSessionId` query parameters are now deleted atomically in a single `setSearchParams` transaction, guaranteeing that the drawer closes immediately.

## 0.13.3 (2026-07-01)

### Fixes

- **Exception Drawer Close Loop** — Fixed an issue where closing the Exception Drawer caused it to instantly reopen. This was resolved by using React state refs inside the main occurrences loader to prevent changes to the active session ID query parameter from triggering full re-query loops.
- **Stable Session Sharing** — Added sync for `?exceptionSessionId` to the browser URL so that selected session timelines inside the drawer are fully stable and shareable.

## 0.13.2 (2026-07-01)

### Fixes

- **Production Plugin Config Crash** — Disabled Webpack code-splitting (chunk limit = 1) to bundle all pages and config components directly into `module.js`. This completely eliminates `ChunkLoadError` issues caused by missing or deleted chunk files (like `732.js`) in production deployments.

## 0.13.1 (2026-07-01)

### Fixes

- **Statically Linked Backend Binaries** — Statically compiled all Linux backend binaries with `CGO_ENABLED=0` to prevent execution failures (like `no such file or directory` or missing dynamic linker) inside minimal Alpine/Debian-slim Docker containers.
- **Dependency Security Fixes** — Upgraded `react-router-dom` to `6.30.4` to remediate the open redirect vulnerability (CVE-2026-40181).

## 0.13.0 (2026-07-01)

### Features

- **Session context switcher** — The Exception Drawer now allows developers to select different sessions (or occurrences) for the same exception hash directly from a Combobox dropdown. This dynamically updates both the metadata and the Session Timeline (breadcrumbs) in-place without losing context.
- **Custom event attribute rendering** — The Session Timeline now parses, extracts, and renders custom event attributes (e.g. from `faro.api.pushEvent`) formatted inline. Events with different attributes are no longer collapsed/deduplicated together to allow tracing distinct event progressions.

### Fixes

- **Console errors panel visibility** — Solved a gap where some console errors logged via Faro's console integration (captured as `kind="exception"` without a stack trace/hash) did not show up in the Top Exceptions table or the Console Errors panel. The panel query was expanded to match both `kind="log"` and `kind="exception"` console errors.
- **Duplicate event grouping** — Timeline breadcrumbs now group consecutive identical events in a row (e.g. repeated resource loads or console errors) and show a count badge (e.g. `(5x)`) to keep the history compact.

## 0.12.3 (2026-06-30)

### Features

- **Watchlist PromQL Performance Optimization** — Constrained queries for the Ops Status Board to only fetch services configured on the watchlist (including rate, error, latency, sparklines, and delta comparisons), preventing expensive cluster-wide Prometheus range queries.
- **UI Render Virtualization** — Limited the Ops Status Board service picker modal list to the top 100 entries to prevent browser stutter and rendering lag on large clusters.

### Fixes

- **React Key Collisions** — Switched service picker list elements to use unique compound keys (`${namespace}/${name}/${environment}`) to solve duplicate key warnings when services exist in multiple environments.
- **Misleading Empty State** — Hidden the "No services configured" empty state illustration if settings or watchlist loading fails with an error.
- **Attention Badge Accuracy** — Decoupled the attention badge calculation from the environment-filtered cards, ensuring critical issues are globally visible regardless of environment filters.
- **AppConfig Robustness** — Ensured `opsWatchlist` defaults to an empty array even if stored settings value is `null` or a non-array value.

## 0.12.2 (2026-06-04)

### Features

- **Inline ops service picker** — Ops Status Board can now add or remove monitored services directly from the dashboard with a Grafana modal, instead of forcing users into plugin configuration.

### Fixes

- **Watchlist diagnostics** — Ops watchlist fetch/save errors now surface a sanitized, contextual error message instead of a generic failure.
- **ServiceInventory test stability** — Reduced unnecessary async wrapping in the favorites toggle test to avoid flaky timing behavior in the full suite.

## 0.12.1 (2026-05-30)

### Fixes

- **Dependency vulnerabilities** — Patched js-cookie (CVE-2026-46625, HIGH), ws (memory disclosure), and qs (DoS) via pnpm overrides. Fixes Grafana plugin validator failure on release.

## 0.12.0 (2026-05-27)

### Features

- **Ops Status Board** — New cross-namespace monitoring page with curated watchlist, auto-rotating grid layout, and at-a-glance health indicators (error rate, latency P95, request rate). Designed for wall-mounted displays in team areas.
- **Shared ops watchlist** — Globally editable watchlist stored in plugin settings via a backend API endpoint. Any authenticated user can add/remove services without admin access. Backend uses service account token to update plugin jsonData on the user's behalf.
- **Centralized Loki support** — Cluster-scoped log queries for environments using a single centralized Loki instance instead of per-namespace datasources.

### Fixes

- **Watchlist security** — Preserve `secureJsonFields` when saving plugin settings to prevent wiping the service account token.
- **Watchlist concurrency** — Mutex-protected read-modify-write cycle prevents concurrent API calls from overwriting each other.
- **React concurrent mode safety** — Network calls moved out of state updater functions to avoid duplicate side effects.

## 0.11.0 (2026-05-19)

### Features

- **Ingress alias support** — Configurable hostname→service mapping makes on-prem (FSS) callers visible when they reach GCP services via nais ingress. For example, mapping `tilgangsmaskin.intern.nav.no → populasjonstilgangskontroll` reveals callers like oppgave and saf that were previously hidden. Backend expands caller queries with additional PromQL `or` clauses for each alias, and resolves aliased hostnames in the service map topology. Frontend adds a key-value editor in plugin settings for managing aliases.

## 0.10.1 (2026-05-07)

### Features

- **Label overrides for non-standard OTel pipelines** — Supports custom Prometheus label names (`serviceNameLabel`, `serviceNamespaceLabel`, `deploymentEnvLabel`) via plugin JSON config. Enables pipelines like Tempo's metrics generator (which emits `service` instead of `service_name`) to work without infrastructure-side relabeling. Both backend queries and frontend PromQL/LogQL are override-aware. Invalid label names are validated with regex and silently dropped. (PR #42, contributed by @m4r1u2)

### Fixes

- **CI fork PR permissions** — Fixed false-positive CI failures on pull requests from forks where the restricted `GITHUB_TOKEN` couldn't post PR comments.
- **Levitate compatibility check** — Changed to advisory mode to avoid blocking PRs on Grafana API type changes that don't affect runtime behavior.

## 0.10.0 (2026-05-06)

### Features

- **Health attention alerts** — Service overview shows an "Attention needed" section when operations or dependencies are degraded. Uses anomaly detection comparing current metrics against a previous period baseline, with both delta-based (2× increase) and absolute (≥5% error rate) thresholds.
- **Causality analysis** — When both operations and dependencies are degraded, the health alert indicates whether errors are likely cascading from downstream dependencies or appear internal. Uses volumetric comparison of dependency error volume against service error volume.
- **Per-page performance route grouping** — The Faro "Per-Page Performance" table now prefers `page_id` (from SDK's `generatePageId`) over raw `page_url` for grouping. Dynamic URL segments (e.g. `/sak/{id}/...`) are automatically normalized when the SDK provides route-level page IDs. Falls back to raw URL when `page_id` is absent. (Closes #41)

## 0.9.1 (2026-04-30)

### Fixes

- **Plugin metadata validation** — Fixed `extensions` field in plugin.json to use the object format (`addedLinks`) expected by Grafana's plugin validator, instead of the array format.

## 0.9.0 (2026-04-30)

### Features

- **Multi-hop service topology** — Explore service connections up to 3 hops deep with a depth control slider. BFS traversal from the focus service discovers transitive callers and dependencies.
- **Per-hop overflow collapsing** — Each hop level independently caps visible nodes (8 at hop 1, 6 at hop 2+), sorted by request rate. Overflow nodes are collapsed into "+N more" placeholders. Cascade hiding removes orphaned deeper nodes.
- **Hub detection and rate-weighted pruning** — Shared infrastructure nodes (e.g., Istio, Wonderwall) with 50+ connections are detected as hubs and displayed with a dashed purple border instead of expanding the full graph. Low-rate edges below 1% of parent rate are pruned to reduce noise.
- **Clickable graph nodes** — Clicking a node in the service topology graph navigates to its detail page. Service nodes go to the service overview; database, messaging, and external nodes go to the dependency page.
- **Node color coding by caller count** — Private dependencies (called by exactly one service) get a lighter blue icon badge, visually distinguishing them from shared services.
- **Fullscreen topology** — Toggle button expands the service topology graph to fill the viewport (Esc to exit).
- **Copy as Mermaid** — Export the current topology graph as a Mermaid diagram to clipboard.

### Fixes

- **Hex-suffixed service name normalization** — Preview/canary deployment names like `nav-dekoratoren-28e8c72f0abdc4109d600c` are normalized to their base name, preventing false self-loop edges.
- **BFS pruning correctness** — Fixed zero-rate pruning bypass, O(E²) infra node detection (now O(1) via pre-computed set), and non-deterministic candidate rate selection (now uses max rate across all edges).
- **Direction-aware BFS** — Callers expand leftward and targets expand rightward, producing a proper layered layout.
- **PromQL escaping** — Uses `promQLEscape` instead of `regexp.QuoteMeta` for correct PromQL regex patterns.
- **Dependency navigation from graph** — Clicking a database/messaging/external node now navigates to the dependency page instead of constructing a broken service URL.
- **ELK layout crash prevention** — Edges are included in the fallback layout pass to prevent crashes on degenerate graphs.

## 0.8.1 (2026-04-29)

### Features

- **Grouped Dependencies tab** — Dependencies are organized into sections (Attention, Databases & Messaging, Internal Services, External Services) with independent sorting per group.
- **Graph node overflow cap** — Service topology graph caps visible nodes at 8 per side, sorted by request rate, with a clickable "+N more" placeholder for overflow.
- **"View all in Dependencies tab" links** — Overview tab shows top-5 per dependency group with links that navigate to the full Dependencies tab.

### Fixes

- **K8s address classification** — Service names like `service.namespace` are now correctly classified as internal instead of external (TLD-aware hostname heuristic).
- **Topology edge completeness** — Outbound topology edges are always supplemented from span metrics, with a dynamic rate window matching the dashboard time range.
- **Favorites redirect preserves params** — Navigating to `/favorites` now preserves existing query params (env, sort, etc.) instead of dropping them.

## 0.8.0 (2026-04-29)

### Features

- **Favorite services ("My Apps")** — Star services to mark them as favorites; toggle the "My Apps" pill to filter the list to just your starred services.
- **Cross-device favorites sync** — Favorites are persisted to Grafana's per-user backend storage via `usePluginUserStorage`, so starred services follow you across browsers and devices. localStorage serves as an instant-render cache.
- **"My Apps" navigation entry** — Dedicated sidebar link that opens the service list pre-filtered to your favorites.
- **"Has errors" filter** — New pill toggle to show only services with active errors (errorRate > 0).
- **SDK/Language filter** — Multi-select dropdown to filter services by telemetry SDK language (Java, Node.js, Go, etc.).
- **URL state persistence** — All filter, sort, pagination, and favorite state is persisted to URL query parameters for shareable deep links.

### Improvements

- **Two-row toolbar layout** — Filters are split into semantic rows: Row 1 (scope) for data filters (search, namespace, environment, SDK, time range) and Row 2 (view options) for display toggles using Grafana-native FilterPill components.
- **PromValue type safety** — Replaced untyped `[2]interface{}` Prometheus value type with a proper struct and custom JSON marshaling, eliminating silent data corruption risks.
- **Shared plugin types** — Extracted `DsRef`, `EnvAwareDs`, `AppPluginSettings` into a shared `types/plugin.ts` module.
- **Context propagation** — Backend health checks now use `context.WithoutCancel` to preserve tracing spans.
- **useServiceData hook** — Extracted data-fetching hooks from ServiceOverview into a dedicated hook for testability.

## 0.7.3 (2026-04-28)

### Features

- **Pageloads & Sessions stats** — New stat panels showing total page loads (from FCP histogram) and unique sessions for the selected time range.

### Improvements

- **Accurate Pageloads count** — Uses FCP histogram bucket instead of generic measurements counter, giving a true per-page-load count.
- **Panel descriptions in plain language** — All tooltip descriptions rewritten to be understandable without prior CWV knowledge.
- **Grafana-native tooltips on bullet charts** — Help icons now use styled Grafana popovers instead of browser title attributes.
- **Per-Page Performance table** — Removed empty Measurements/TTFB columns; table auto-sizes to fill available width.
- **Top Exceptions table** — Error column auto-expands to fill table; count columns stay narrow.
- **Insights row layout** — CWV Rating, Pageloads, and Sessions panels equal width (15% each); Navigation Type gets remaining space.

## 0.7.2 (2026-04-28)

### Features

- **Pageloads & Sessions stats** — New stat panels in the Frontend tab showing total page loads (Mimir) and unique sessions (Loki) for the selected time range.

### Improvements

- **Responsive toolbar layout** — Filter inputs use flex-grow to fill available horizontal space (min 160px, max 320px) instead of fixed widths.
- **No layout shift on multi-select expand** — Toolbar uses absolute positioning so expanded pills overlap the table instead of pushing it down.
- **Top Exceptions table column widths** — Error column gets most of the horizontal space; count columns are narrow and right-aligned.

## 0.7.0 (2026-04-28)

### Breaking Changes

- **Mimir-first architecture** — Removed all Loki fallback logic. The Frontend tab now requires the Alloy Faro histogram pipeline in Mimir. Legacy gauge-based and raw-Loki-only pipelines are no longer supported.

### Features

- **CWV Rating panel** — Stat panels showing "% Good" for each Core Web Vital (LCP, FCP, CLS, INP, TTFB) computed from histogram bucket boundaries.
- **Browser Volume panel** — Pie chart showing measurement distribution by browser (Loki-based, visible when Loki data available).
- **Per-Page Performance table** — Top 20 pages ranked by traffic volume with color-coded vital thresholds.
- **Console Errors panel** — Shows top `console.error` messages when apps enable Faro `ConsoleInstrumentation`.
- **Traffic timeseries** — Combined page loads, error rate, and session starts over time.
- **Faro Kind filter in Logs tab** — When "Include browser telemetry" is on, filter by kind (Exceptions, Console Logs, Measurements, Events) using the indexed `kind` stream label.
- **Exception drill-down** — Clicking an exception navigates to the Logs tab pre-filtered to `kind=exception` with Faro telemetry enabled and the correct environment datasource.

### Improvements

- **Simplified capabilities model** — Reduced to `{ available, hasLoki, vitals, errorRate }`, removing multi-source adapter abstraction.
- **1-hour default time range** — Frontend tab defaults to `now-1h` instead of inheriting global range (prevents Loki timeouts).
- **Centralized metric definitions** — All Mimir counter/histogram names and Loki field names in `otelconfig.ts`.
- **Section builder architecture** — Six composable builders (`buildCwvRatingSection`, `buildHistogramSection`, `buildPerPageSection`, `buildErrorsSection`, `buildSupportSection`, `buildTrafficSection`).
- **Errors section layout** — Split cramped 4-panel row into two rows: Exceptions (types + messages) and Browsers (breakdown + volume).

### Bug Fixes

- **CWV Rating "Bar charts require a string field"** — Switched from barchart (needs string x-axis) to stat panels.
- **Browser Volume "Value #volume"** — Range queries resolve `legendFormat` templates; instant queries don't.
- **hasLoki detection** — Widened detection window from 1h to 6h to avoid false negatives on low-traffic apps.
- **Exception links** — Fixed navigation for empty namespaces (uses `_` placeholder), passes environment and Faro filters.

### Removed

- Legacy Alloy gauge pipeline support
- Loki-only fallback path (`adapter.ts`, multi-source resolution)
- `metricsSource` field from backend response

## 0.6.2 (2026-04-27)

### Bug Fixes

- **Multi-environment filtering in topology graph** — Backend now supports comma-separated environment values using PromQL regex matching (`=~"prod|prod-fss"`). The namespace page service map and dependencies correctly filter when multiple environments are selected.
- **E2E test reliability** — Replaced `networkidle` wait with `load` state in test fixtures to prevent timeouts from Grafana's persistent WebSocket connections.

## 0.6.1 (2026-04-27)

### Bug Fixes

- **Service topology graph respects environment filter** — The service map and namespace dependencies on the namespace page were ignoring the multi-select environment filter. Now correctly passes the selected environment to the backend when a single environment is chosen.

## 0.6.0 (2026-04-26)

### Features

- **Status Board** — New TV-friendly dashboard view showing all services in a namespace as color-coded health cards (green/yellow/red/gray). Auto-refreshes, tracks last-seen services, and auto-paginates through cards when they exceed the viewport. Accessible from the namespace page via "Status Board" button. (#29)
- **Card Size Selector** — Status Board supports S/M/L card sizes to fit more or fewer services on screen. Small cards show name + status only; large cards include sparkline trends.
- **Auto-Pagination** — Status Board automatically rotates through pages at a configurable interval (10s/30s/60s) when services exceed the viewport, perfect for wall-mounted displays.
- **Multi-Select Environment Filter** — All pages (Services, Namespace, Status Board) now support filtering by multiple environments simultaneously using a multi-select combobox.
- **Mise E2E Tasks** — Added `mise run e2e` and `mise run e2e:ui` for running Playwright tests locally with a health check gate.

### Improvements

- **Explicit Route Redirect** — Plugin root URL now properly redirects to `/services` instead of rendering the page at the root path.
- **Sandbox-Resilient Build** — `mise run all` now works in sandboxed environments (macOS sandbox, CI) by auto-detecting inaccessible parent configs and gracefully falling back for coverage/mage tools.
- **Quieter Build Output** — `mise run all` shows brief `✓ passed` on success and full output only on failure.

### Bug Fixes

- **Grid Layout** — Fixed Status Board showing 1 card per row when data loaded asynchronously (ResizeObserver now attached via callback ref outside conditional rendering).
- **Filter Trap** — Fixed issue where hiding the environment filter when only one environment existed could trap users with an active but invisible filter.
- **E2E Test Stability** — Tests now use Playwright `.or()` locator chains to handle Grafana nav rendering differences across versions 12–13.

## 0.5.4 (2026-04-26)

### Features

- **Namespace Alert Rules** — Namespace page now shows Prometheus alerting rules for the team, fetched from the Mimir ruler API. Displays firing, pending, and inactive rules with severity badges, instance counts, and relative timestamps. Source is labeled "Prometheus Alerts" so teams know where rules come from.
- **Cross-Cluster Alert Deduplication** — Same alert rule defined in multiple clusters (e.g., `dev` and `dev-fss`) is automatically merged into a single entry with combined instance counts and the earliest active timestamp.

### Improvements

- **Alerts 2-Column Layout** — Alert cards use a responsive two-column grid with subtle state-colored backgrounds instead of heavy bordered boxes.
- **Stat Panels De-Boxed** — Namespace stats (Services, Rate, Errors, Healthy) now render without borders for a cleaner look.
- **Inactive Rules Collapsed** — Inactive alert rules are collapsed by default with an expandable toggle to reduce noise.

### Bug Fixes

- **Mimir Compatibility** — Removed `type=alerting` query parameter from ruler API call (unsupported by some Mimir versions); filter alerting rules client-side instead.
- **Ruler File Path Extraction** — Fixed namespace extraction from NAIS ruler file paths (`{cluster}/{namespace}/{name}/{uuid}` format — was incorrectly extracting the cluster segment).
- **Unknown Alert States** — Unknown/unexpected alert states now sort to the bottom instead of being treated as highest priority (firing).
- **Invalid Date Handling** — Invalid or future `activeSince` timestamps now display the raw ISO string instead of "NaNd ago".
- **Description Merge** — Alert description field is now properly merged during cross-cluster deduplication.

## 0.5.3 (2026-04-25)

### Features

- **Dependencies Tab with Callers & Dependencies** — Dependencies tab now shows both Callers (upstream services) and Dependencies (downstream databases/APIs/services) in two sortable sections with independent sort state.
- **Dependencies Panel on Overview** — Compact dependencies table on the service Overview tab showing outbound dependencies alongside the Callers table.
- **Clickable Operations → Traces** — Operation rows on the Overview tab are now clickable, navigating directly to the Traces tab pre-filtered by span name and span kind.
- **Outbound Calls Section** — Operations tab (formerly "Server") now includes an "Outbound Calls (Client)" section showing HTTP client endpoints (excluding database spans which have their own section).
- **Spanmetrics Topology Fallback** — Service topology graph now falls back to spanmetrics-based edges when service graph metrics are unavailable, ensuring graphs render in all environments.
- **Namespace Health Indicators** — Enhanced namespace page with health status indicators and a "Needs Attention" section highlighting services with elevated errors or latency.

### Improvements

- **"Server" → "Operations" Tab Rename** — Renamed the Server tab to Operations to better reflect its expanded scope (inbound + outbound endpoints).
- **"Connected Services" → "Callers" Naming** — Unified terminology across the app: upstream services are "Callers", downstream services are "Dependencies".
- **Tempo Availability Gating** — "View traces" actions on the Overview tab are only shown when Tempo is available, preventing dead-end clicks.
- **UI Polish** — Replaced emoji indicators with themed components, tightened layout spacing, improved namespace table and NeedsAttention card styling.
- **Documentation** — Added design philosophy and visualization guide docs; updated all references for the Operations/Callers rename.

### Bug Fixes

- **Topology Graph for prod-fss Services** — Fixed topology graph not rendering for services in `prod-fss` and other environments without service graph metrics.
- **Service Graph Scoping** — Scoped service graph queries by service for per-service topology; skip namespace filter for per-service views.
- **Environment Edge Filtering** — Service graph edges now properly filter by environment.
- **Dependency Error Rate Display** — Fixed dependency error rate showing as 10000% instead of 100%.
- **RED Panel No Data** — Prevented RED panels from showing "No Data" during capability loading phase.

## 0.5.2 (2026-04-23)

### Features

- **Bullet Graph Visualizations** — New compact bullet graph cards for all five Core Web Vitals (LCP, FCP, CLS, INP, TTFB) with qualitative color bands and threshold markers. Works for both Mimir and Loki data sources.
- **Loki Vital Value Extraction** — Backend now queries Loki for individual vital values using weighted mean aggregation, enabling bullet graphs for Faro-in-Loki services.
- **Environment Dropdown in Config** — Plugin config page auto-detects available environments from Mimir data and presents them as a dropdown instead of a free-text input.
- **Browser Filter Dropdown** — Frontend tab includes a Browser variable selector (Chrome, Firefox, Safari, Edge, Opera, Samsung Internet) for filtering Web Vitals by browser.
- **Logs Tab Enhancements** — Pod filter, severity stream labels, and Faro log toggle added to the Logs tab.

### Bug Fixes

- **Environment Datasource Routing** — `MimirWebVitalsPanels` now receives the `environment` prop so it queries the correct Loki/Tempo datasources instead of always falling back to the default.
- **Provisioning Config Keys** — Updated provisioning file from `dev-gcp`/`prod-gcp` to match actual `k8s_cluster_name` values (`dev`, `dev-fss`, `prod`, `prod-fss`), preventing config reset on every deploy.
- **Weighted Mean for Loki Vitals** — Changed Loki vital queries from `avg(avg_over_time)` to `sum(sum_over_time) / sum(count_over_time)` for proper weighted aggregation matching Scene panel values.
- **Bullet Graph Layout Shift** — Rating text ("Needs improvement") now renders on its own line to prevent inconsistent card heights.
- **Controls Above Bullet Charts** — Bullet graphs are now embedded inside the EmbeddedScene so Browser/time controls render above them.
- **Topology Graph with Environment Filter** — Service topology graph now renders correctly regardless of environment filter selection.
- **Logs Severity Filter** — Moved severity filter from Loki stream selector to label filter pipeline and expanded matching to handle all `detected_level` case variants (e.g. `Error`, `ERROR`, `error`).

### Improvements

- **Frontend Tab Layout** — Improved table readability and dashboard layout for the Frontend tab.

## 0.5.1 (2026-04-23)

### Bug Fixes

- **E2E: Remove Stale Service Map Test** — Remove `service map page should render` e2e test that referenced the deleted `ROUTES.ServiceMap` constant (page removed in v0.5.0).
- **README: Absolute Links** — Convert all relative links in README.md to absolute GitHub URLs to satisfy the Grafana plugin validator for marketplace publishing.

## 0.5.0 (2026-04-23)

### Features

- **Namespace Overview Page** — New team/namespace page (`/namespaces/{ns}`) with aggregate stats tiles, service topology graph, services table with search/pagination, and external dependencies table with pagination. Navigate from any namespace link in ServiceInventory.
- **Server-Side Namespace Filtering** — Backend `/service-map` and new `/namespace-dependencies` endpoints filter by namespace using a service→namespace mapping from spanmetrics. Eliminates client-side filtering inaccuracies.
- **Service Topology for Large Teams** — Removed the 20-service cap on namespace topology graphs. ELK layout handles up to 300 nodes with zoom/pan controls.
- **Dependency Enrichment** — Database and messaging dependencies display enriched names like `postgresql (10.0.0.1)` or `kafka (brokers.example.com)` using `db_system` and `messaging_system` labels from Tempo's service graph metrics. No cross-fetching from spanmetrics needed.
- **Mermaid Export** — Copy button on the namespace topology graph exports the service graph as a Mermaid flowchart. Node shapes match types: cylinders for databases, hexagons for messaging, stadiums for external services.
- **Frontend Tab: Per-Page Performance** — New table grouping all five Core Web Vitals (LCP, FCP, CLS, INP, TTFB) by page URL with threshold-colored cells and measurement counts.
- **Frontend Tab: Console Errors** — New panel showing most frequent `console.error` messages from Faro logs, replacing the noisy Top Events panel.
- **Frontend Tab: Enhanced Exceptions** — Top Exceptions expanded to top 20 with full error messages, "Sessions Affected" column, and click-through Explore links to Loki.
- **Frontend Tab: Threshold Bands** — Green/yellow/red Web Vitals reference zones on INP and CLS time series panels (both Loki and Mimir paths).

### Improvements

- **Consistent Back Button** — Shared `BackButton` component across ServiceOverview, DependencyDetail, and NamespaceOverview pages.
- **Messaging System Classification** — `classifyDependency` now correctly distinguishes between Kafka, RabbitMQ, JMS, and generic messaging types, matching frontend icon rendering.
- **Shared Components** — Extracted `PageHeader`, `DataState`, section styles, and option helpers into reusable modules.
- **Environment Column** — Namespace services table shows environment column when multiple environments exist (hidden when env filter is active).
- **Framework Badge Styling** — Lighter weight (`font-weight: 400`) and subtle rounding (`border-radius: 4px`) for framework badges. Unknown frameworks now render a gray fallback badge instead of being hidden.
- **Canvas Layout** — Namespace page uses `PageLayoutType.Canvas` for a clean, headerless layout matching the service detail page.

### Bug Fixes

- **Service Node Misclassification** — Fix services receiving Kafka traffic being rendered as messaging infrastructure nodes. A node that also initiates calls is a service, not a broker.
- **Service Filter on Team Page** — Fix search filter not matching services and column alignment issues in the namespace services table.
- **Empty Service Topology** — Fix topology graph rendering as blank on namespace page. Root cause: ReactFlow `height: 100%` resolved to 0px when parent had no explicit height. Fixed by wrapping graph in container with explicit height.
- **Environment Dropdown Stuck** — Fix environment dropdown disappearing after selection. Now uses a separate unfiltered fetch for environment discovery, and always shows dropdown when a filter is active.
- **Sidecar Filtering** — Namespace page now hides known infrastructure sidecars (wonderwall, texas) consistent with ServiceInventory default behavior.
- **Empty Namespace Navigation** — Namespace links in ServiceInventory only render as clickable when namespace is non-empty, preventing broken route navigation.
- **Frontend Tab: Loki Aggregation** — Fix incorrect averages: `avg(avg_over_time(...))` computed average-per-stream then averaged those (wrong when streams have unequal sample counts). Now uses weighted mean via `sum(sum_over_time(...))/sum(count_over_time(...))`.
- **Frontend Tab: Rating Pie Chart** — Fix "Value #A" labels on the Web Vitals rating distribution pie chart. Now correctly shows "good", "needs-improvement", "poor".
- **Frontend Tab: Loki Query Parse Error** — Fix `count_over_time` failing with `unwrap` operator. The `unwrap` stage is now only applied to `sum_over_time`.
- **Frontend Tab: Explore Links** — Fix broken "Explore in Loki" links on exceptions table that produced duplicate `left=` URL parameters.

### Removed

- **Global Service Map** — Removed the global Service Map page which crashed browsers by loading all ~4000 services into a single graph. Per-service and per-namespace service maps remain functional. See [#22](https://github.com/nais/grafana-apm-app/issues/22) for the planned clustered replacement.

### Chores

- Add `.DS_Store` and `.vscode` to `.gitignore`.

## 0.4.4 (2026-04-22)

### Improvements

- **Backend Code Quality**: Extract shared request parsing (`reqparse.go`) and metric math helpers (`metricmath.go`), eliminating duplicated patterns across handlers.
- **Dependencies Module**: Split monolithic `dependencies.go` (1244 lines) into 5 focused files: handlers, classification, detail queries, connected services, and shared helpers.
- **Services Module**: Extract `aggregateServiceResults` from `fetchServiceSummaries`, splitting a 408-line function into two cohesive halves.
- **ServiceOverview Extraction**: Extract `buildServiceScene` (scene builder) and `OverviewTab` (operations, topology, connected services) — reduces ServiceOverview.tsx from 960 to 488 lines.
- **Parallel Query Pattern**: Convert manual WaitGroup/channel pattern in `queryConnectedServices` to use the existing `runInstantQueries` helper.

## 0.4.3 (2026-04-22)

### Features

- **Combobox Migration**: Migrate all 15 deprecated `Select` components to `Combobox`/`MultiCombobox` from `@grafana/ui` v12.4.2.
- **Release Script**: Add `mise run release` task for automated version bumping, CHANGELOG stamping, build verification, and git tagging.

### Bug Fixes

- **Exemplar Links**: Route trace exemplar links to the environment-resolved Tempo datasource instead of hardcoded dev Tempo.
- **Dependency Environment Filter**: Filter service graph dependencies by selected environment — prevents pre-prod dependencies from appearing when a prod environment is selected.
- **Kafka Consumer Dependencies**: Exclude Kafka consumer services (e.g., downstream subscribers) from the dependencies list — they are not dependencies of the producer and now only appear in Connected Services.
- **Connected Services Environment**: Pass environment filter to connected services queries so inbound/outbound services are scoped to the selected cluster.

## 0.4.2 (2026-04-22)

### Features

- **Panel Descriptions**: RED panels on ServiceOverview now include help text explaining what each panel measures (e.g., inbound SERVER requests vs all span kinds).

### Bug Fixes

- **Dependency Detail**: Fix dependency metrics showing zeros for addresses with standard ports (e.g., `idporten.no:443`) — queries now use regex matching to handle both normalized and raw address forms.
- **Service Detail Navigation**: Fix error when clicking connected services that have no namespace — the `_` wildcard placeholder was converted to empty string, producing double-slashes in API URLs that Grafana couldn't route to the plugin backend.
- **RED Panel Flashing**: Reduce unnecessary panel re-renders on ServiceOverview by stabilizing Scenes `useMemo` dependencies — destructure datasource/metric objects into primitive strings to prevent object identity changes from recreating the entire `EmbeddedScene`.

## 0.4.1 (2026-04-22)

### Features

- **Framework Detection**: Use `telemetry_sdk_language` from span metrics as fallback when framework-specific metric probing doesn't match. Adds Python, .NET, Java, and Go badges for services without framework-specific runtime metrics. Specific detection (Spring Boot, Ktor) still takes priority. Closes #20.

### Bug Fixes

- **Traces Tab**: Fix bad request when filtering by duration — move `duration >=`/`<=` inside the TraceQL span selector instead of using unsupported pipeline syntax.
- **Service Inventory**: Shrink Type column to fit badge width instead of fixed 120px. Remove Faro badge to prevent row wrapping.

## 0.4.0 (2026-04-22)

### Features

- **Dependency Detail**: Database dependencies now show target database and operation (`db_name · db_operation`) in a dedicated "Target" column in the operations table.
- **Dependency Detail**: Messaging dependencies now show topic names (`messaging.destination.name`) in a dedicated "Topic" column in the operations table.
- **Exemplars**: RED panels on ServiceOverview and DependencyDetail pages now display exemplar data points — click to jump directly to the originating trace.
- **Service Map**: Explicit handling of `virtual_node` connection type for external dependency classification.

### Bug Fixes

- **Service Inventory**: Services with only CLIENT, PRODUCER, or CONSUMER spans (Kafka consumers, batch jobs, event-driven workers) now appear in the inventory with full RED metrics and sparklines. Previously, only services with SERVER spans were discovered. Server-based services are unchanged — they continue using SERVER-only metrics for clean inbound request data.

## 0.3.1 (2026-04-22)

### Bug Fixes

- **Dependency Detail**: Fix query parse error for dependency names containing dots (e.g., `dmv18-scan.adeo.no`) — PromQL regex matchers now correctly double-escape special characters.

### CI

- Strip plugin signature in E2E tests so Grafana loads the plugin on localhost.
- Generate release notes from CHANGELOG.md instead of Grafana boilerplate.

## 0.3.0 (2026-04-22)

### Features

- **Sidecar Filtering**: Sidecars (wonderwall, texas) are identified and hidden by default in the service inventory with a toggle to show them. Sidecar badges and visual de-emphasis in connected services and service map views.
- **Framework Detection**: Improved detection for Go (`go_info`) and Spring Boot (`application_started_time_seconds`). Node.js correctly takes priority over Go when both metrics are present. Sidecar containers (wonderwall, texas) are excluded from Go detection — they share the pod's `app` label, which falsely marked Node.js apps as Go.
- **Environment Grouping**: When filtering by namespace, services are automatically grouped by environment with visual section headers — helps teams see their apps across prod/dev/staging.
- **Namespace Column**: Namespace column is automatically hidden when a namespace filter is active to reduce redundancy.
- **Percentile Help**: Added tooltip on the Server tab explaining how histogram bucket boundaries affect P95/P99 accuracy, with link to configuration docs.

### Bug Fixes

- **Error Rate Calculation**: 5 call sites used raw division instead of `calculateErrorRate()`, risking NaN/Inf display. All now use the canonical helper with validation and clamping.
- **Input Sanitization**: Defense-in-depth `MustSanitizeLabel()` applied to environment parameter in frontend metrics handler.
- **Client-side Filtering**: Fixed filtering on Dependencies and ServiceInventory pages not responding to user input.
- **Dependency Detail**: Fixed external services missing data — address normalization, deduplication, and P95 calculation for external dependencies.
- **Trace Search**: Search `http.route` in addition to span name for trace lookups, improving hit rate.
- **Error Sparklines**: Error sparkline column now renders actual time-series charts instead of static bars.
- **Pod Count**: Runtime pod count deduplicates by instance/pod, fixing inflated container counts on the Server tab.
- **Duplicate Rows**: Fixed service list showing duplicate rows from inconsistent metric labels.
- **URL Sanitization**: Fixed corrupted environment params from browser history on all pages.
- **Navigation & Auth**: Fixed critical bugs in navigation, auth header forwarding, and regex patterns.
- **External Dependencies**: Include external domains in global dependencies list.

### Refactoring

- Consolidated duplicate model types across packages and reduced verbose code patterns.
- Added critical tests for API contract, metric math, and model serialization.

### Documentation

- Comprehensive metrics pipeline and operator configuration guide.
- Documented all metrics, traces, and log data with origins.

### CI

- Fix release workflow to publish instead of leaving as draft.
- E2E workflow only starts Grafana, not full LGTM stack.
- Heatmap height and topology layout improvements.

## 0.2.1 (2026-04-22)

### Features

- **Auth**: Auto-managed service account via Grafana's `externalServiceAccounts` feature (Grafana 10.3+). When enabled, the plugin authenticates automatically with zero configuration — no manual token needed.
- **Auth**: 3-tier auth fallback: auto-managed token → manual service account token → forwarded user headers
- **Auth**: Added IAM permissions (`datasources:query`) to `plugin.json` for zero-config auth
- **Config Page**: Updated authentication section to document both auto-managed and manual token options
- **Environment**: Dependencies page now respects the selected environment filter — spanmetrics queries filter by `k8s_cluster_name` for accurate per-environment dependency data
- **Environment**: Added environment dropdown to the global Dependencies page
- **Environment**: Environment selection is preserved through dependency navigation (list → detail → upstream services → back)
- **Environment**: Faro frontend detection now uses environment-specific Loki when an environment filter is active
- **Environment**: Info banner on Traces/Logs tabs when no environment is selected and per-environment datasources are configured

### Utilities

- **Datasources**: New `useConfiguredEnvironments()` and `useHasEnvironmentOverrides()` hooks for reading plugin environment config

## 0.2.0 (2026-04-21)

### Features

- **Auth**: Service account token support for deployments behind OAuth2 proxies (Wonderwall/Nais). The plugin backend now authenticates internal datasource proxy calls using a Grafana service account token stored in `secureJsonData`, fixing 401 errors when Grafana runs behind an OAuth2 sidecar.
- **Config Page**: Prerequisites alert with links to OTel Collector spanmetrics and servicegraph connector documentation
- **Config Page**: Detailed descriptions for all configuration sections — datasource purposes, environment overrides, authentication, and auto-detection behavior
- **Config Page**: Warning alert when auto-detect finds no span metrics, with troubleshooting guidance
- **Config Page**: Detection results now show Tempo, Loki, and service graph connectivity status

### Bug Fixes

- **Auth**: Centralized `applyAuth()` on PrometheusClient ensures consistent auth across all internal API calls (queries, label lookups, health checks)
- **Auth**: Preserve `X-Grafana-Org-Id` header when using service account token to maintain correct org context

### CI

- Remove invalid `publish: true` input from release workflow
- Update e2e action versions (e2e-version v1.2.1, wait-for-grafana v1.0.2, upload-report-artifacts v1.0.1)
- Remove deprecated `config.apps` usage from datasource config resolution
- Fix golangci-lint revive false positive for standard `plugin` package name

## 0.1.2 (2026-04-21)

### Bug Fixes

- **Endpoint Table**: Fix truncated endpoint names showing identical text via colgroup column widths and 2-line clamp
- **Heatmap**: Duration distribution now syncs with page time picker by merging into shared EmbeddedScene
- **Heatmap**: Add minInterval to prevent sparse data in short time windows
- **Dependencies**: Dependency detail charts fall back to spanmetrics (`server_address`, `http_host`) when service graph metrics are unavailable
- **Traces**: Traces tab now uses shared page time range instead of hardcoded 1-hour window
- **Dependencies**: Fix impact column bar+label alignment in dependencies tab
- **Backend**: Use localhost for internal API calls and propagate auth headers for production proxy

### Features

- **Service Inventory**: Add time range picker dropdown to toolbar
- **Config**: Datasource configuration uses dropdown selectors populated from Grafana API, with auto-detection of default/sole datasources and environment override patterns
- **Tooling**: Convert Makefile to mise task runner with hierarchical tasks (`frontend:*`, `backend:*`, `deploy:*`)
- **Tooling**: Add local deployment tasks (`deploy`, `deploy:up`, `deploy:down`, `deploy:restart`, `deploy:logs`, `deploy:status`, `deploy:demo`)
- **Tooling**: Pre-commit hook runs full check+test+build pipeline via `mise run all`

### CI

- Fix workflow file issue: use `env` context instead of `secrets` in step-level `if:` condition
- Add gitleaks secrets scan and plugin.json validation to mise check pipeline
- Backend tests now enforce Go coverage ≥ 10% threshold (matching CI)

## 0.1.1 (2026-04-21)

### Bug Fixes

- **Cache**: Hard limit prevents unbounded memory growth past maxSize
- **Cache**: Add org ID to cache keys to prevent cross-tenant data leaks
- **Cache**: Lazy cleanup of expired entries on read
- **PromQL**: Use `regexp.QuoteMeta` instead of `url.PathEscape` for regex escaping

### Performance

- **Runtime**: Limit query fan-out to 3 concurrent categories (~30 peak queries)
- **Frontend**: Vitals queries run in parallel instead of sequentially
- **Service Graph**: ELK layout skipped for graphs over 300 nodes (grid fallback)
- **Service Graph**: Proper `setTimeout` cleanup in useEffect
- **Health Checks**: Shared HTTP client for connection reuse
- **Health Checks**: Drain response body to prevent connection pool leaks
- **Capabilities**: Reduce detached context timeout from 60s to 30s

### CI

- Bump grafana/plugin-actions to v1.1.0
- Bump magefile/mage-action to v4.0.0
- Bump actions/setup-node to v6.4.0

## 0.1.0 (2026-04-20)

First public release of Nais APM — a self-hosted APM plugin
powered by OpenTelemetry data from Mimir, Loki, and Tempo.

### Features

- **Service Inventory** — auto-discovered table of all instrumented services with health
  sparklines, SDK language icons, namespace/environment filters, search, sort, and pagination
- **Service Overview** — per-service RED panels (Rate, Errors, Duration) with configurable
  percentile selectors (P50/P75/P90/P95/P99) and exemplar overlays
- **Operations Breakdown** — top operations table and duration distribution histogram
- **Service Map** — topology graph showing inter-service dependencies with error highlighting,
  collapsible dormant nodes, and ELK.js auto-layout
- **Dependencies** — downstream dependency inventory with impact scoring, upstream callers,
  per-dependency operation breakdown, and RED time-series panels on detail pages
- **Connected Services** — inbound/outbound service tables on service detail pages
- **Cross-Signal Navigation** — click any data point to jump to correlated traces or logs
  in Grafana Explore; clickable endpoint rows navigate to filtered traces
- **Trace Exploration** — search and browse traces scoped to a service with status filtering
- **Log Exploration** — service-scoped log viewer with severity filtering, text search,
  and log volume histogram
- **Server Tab** — HTTP/gRPC/database endpoint breakdown with protocol-specific metadata
- **Frontend Tab** — Core Web Vitals (LCP, FCP, CLS, INP, TTFB), P75 trends, per-page
  performance table, browser breakdown, and duration histogram for Faro-instrumented services
- **Runtime Tab** — container resource utilization, Go runtime metrics, JVM memory pools,
  GC types, and thread counts
- **Auto-Detection** — automatically discovers span metric names, duration units, and
  available capabilities from your data
- **Capability-Driven UI** — tabs and panels only appear when the underlying data exists
- **Environment Filter** — filter all views by deployment environment
- **Deep Linking** — tab selection and filter state persisted in URL via `?tab=` params

### Security

- PromQL label value sanitization on both backend (Go) and frontend (TypeScript)
- Input validation with allowlists on all user-controlled query parameters
- TraceQL query escaping for service names and search terms
- HTTP method enforcement on all API endpoints
- Bounded response reading (10 MB limit) to prevent OOM from large Prometheus responses
- Security vulnerability patches for transitive dependencies (protobufjs, dompurify,
  immutable, serialize-javascript)

### Technical

- Go backend using Grafana Plugin SDK with datasource proxy integration
- React frontend using @grafana/scenes for panel rendering
- Grafana >= 12.0.0 required (tested on 12.0–12.4 and 13.0–13.1)
- E2E tests across 6 Grafana versions including Grafana 13 carousel overlay handling
- CI pipeline with plugin validator, Playwright reports published to GitHub Pages
- Apache 2.0 license
