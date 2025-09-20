# TODOs

## Priority Index

### Priority Levels
- **P1 (CRITICAL)**: Must fix immediately - affects reliability, security, or blocks development
- **P2 (HIGH)**: Should fix soon - performance issues, maintainability problems
- **P3 (MEDIUM)**: Important improvements - architecture enhancements, scalability
- **P4 (LOW)**: Nice to have - code quality, technical debt, optimizations

### Priority Level Distribution
- **P1 Items**: 3 critical issues remaining (test coverage, code duplication, conflict resolution) - 2 completed
- **P2 Items**: 11 high priority tasks remaining (performance, refactoring, backup/restore) - 5 completed  
- **P3 Items**: 18 medium priority tasks (architecture, patterns, GUI/CLI enhancements, job management)
- **P4 Items**: 12 low priority tasks (code quality, dependencies, feature parity, advanced features)

---

## PRIORITIZED TODO ITEMS

### P1: CRITICAL (Reliability & Security)

- [ ] **[P1] Increase Test Coverage to Minimum 60%**
  - API package: Currently 0.3% → Target 80%
  - App package: Currently 0% → Target 70%
  - Database package: Currently 6.6% → Target 75%
  - Add integration tests for pull/push workflows
  - **Impact**: Prevents regression bugs and ensures reliability

- [ ] **[P1] Extract HTTP Client Wrapper**
  - Eliminate 28 instances of duplicate authorization header setting
  - Eliminate 27 instances of duplicate status code checking
  - Create centralized `doRequest()` method with error handling
  - **Impact**: Reduces code duplication by ~50% in API package

- [ ] **[P1] Conflict Resolution Strategy**
  - Define strategy for handling sync conflicts (prompt user, last-write-wins, etc.)
  - Implement conflict detection in push operations
  - Add user interface for conflict resolution
  - **Impact**: Prevents data loss and improves reliability

- [x] **[P1] Webhook Signature Verification** ✅ COMPLETED
  - ✅ Implemented HMAC-SHA256 signature verification for incoming webhooks
  - ✅ Added configuration for webhook secret keys
  - ✅ Reject unsigned or invalid webhook requests
  - ✅ Added timestamp validation to prevent replay attacks
  - **Completed**: Created `app/server/webhook_security.go`
  - **Impact**: Critical security enhancement to prevent webhook spoofing

- [x] **[P1] Audit Logging System** ✅ COMPLETED
  - ✅ Created comprehensive audit trail for all server operations
  - ✅ Log all API calls, database changes, and webhook events
  - ✅ Include user, timestamp, operation type, and outcome
  - ✅ Implemented log rotation and retention policies
  - ✅ Added async logging with queue for performance
  - **Completed**: Created `app/audit/audit_log.go`
  - **Impact**: Essential for compliance, debugging, and security forensics

### P2: HIGH (Performance & Maintainability)

- [ ] **[P2] Add HTTP Client Optimizations**
  - Configure connection pooling (MaxIdleConns: 100, MaxIdleConnsPerHost: 10)
  - Set request timeouts (30 seconds default)
  - Implement retry logic for transient failures
  - **Impact**: Improves API call performance and reliability

- [ ] **[P2] Batch Database Operations**
  - Replace individual CREATE TABLE statements in loops with batch operations
  - Use prepared statements for bulk INSERTs
  - Implement transaction batching for account/checkin updates
  - **Impact**: 3-5x performance improvement for large data operations

- [ ] **[P2] Refactor Long Functions**
  - Break down gui/gui.go functions (currently up to 178 lines)
  - Split database/database.go ValidateSchema (>100 lines)
  - Target maximum 50 lines per function
  - **Impact**: Improves code readability and testability

- [ ] **[P2] Fix Deep Nesting Issues**
  - Reduce 5-level nesting in gui/gui.go:229-233 using early returns
  - Extract complex conditionals into separate methods
  - Use guard clauses to reduce indentation
  - **Impact**: Improves code readability and reduces cognitive complexity

- [ ] **[P2] Dry Run Mode**
  - Add `--dry-run` flag to push command to preview changes before applying
  - Show what would be changed without making actual API calls
  - **Impact**: Improves safety and confidence in push operations

- [ ] **[P2] Enhanced Webhook Triggers**
  - Implement system to trigger actions directly from incoming webhooks
  - Action configuration should support `source: webhook`, `object: <struct_type>`, `event: <Create|Update|Delete>`
  - **Impact**: Enables real-time automation and reduces manual intervention

## SERVER AUTOMATION & SCHEDULING

- [x] **[P2] Scheduled Sync Functionality** ✅ COMPLETED
  - ✅ Implemented cron-based scheduling for automatic pull/push operations
  - ✅ Support flexible cron expressions (e.g., "*/15 * * * *" for every 15 minutes)
  - ✅ Allow different schedules for different sync types (accounts, checkins, routes)
  - ✅ Added timezone support for schedule configuration
  - ✅ Retry mechanism with exponential backoff
  - **Completed**: Created `app/server/scheduler.go`
  - **Impact**: Enables hands-free data synchronization

- [x] **[P2] Data Validation & Health Checks** ✅ COMPLETED
  - ✅ Implemented pre-sync validation to ensure data integrity
  - ✅ Check API connectivity and authentication before operations
  - ✅ Validate database schema compatibility
  - ✅ Health check endpoint for external monitoring
  - ✅ Component health monitoring (database, API, disk, memory)
  - **Completed**: Created `app/server/health_check.go`
  - **Impact**: Prevents corrupt data and failed sync operations

- [x] **[P2] Server Monitoring & Metrics** ✅ COMPLETED
  - ✅ Track sync operation success/failure rates
  - ✅ Monitor API response times and database query performance
  - ✅ Collect memory and CPU usage metrics
  - ✅ Implement health check endpoints for external monitoring
  - ✅ Latency tracking with percentiles (P50, P95, P99)
  - ✅ HTTP metrics endpoint for monitoring integration
  - **Completed**: Created `app/server/metrics.go`
  - **Impact**: Enables proactive issue detection and performance optimization

- [ ] **[P2] Backup & Restore System**
  - Automatic database backup before critical operations
  - Point-in-time recovery capability
  - Compressed backup storage with rotation
  - CLI and GUI interfaces for backup management
  - **Impact**: Protects against data loss and enables rollback

- [x] **[P2] Retry Mechanism with Exponential Backoff** ✅ COMPLETED
  - ✅ Implemented intelligent retry for failed sync operations
  - ✅ Exponential backoff to prevent thundering herd
  - ✅ Configurable max retries and timeout periods
  - ✅ Integrated into scheduler for automatic job retries
  - **Completed**: Integrated into `app/server/scheduler.go`
  - **Impact**: Improves reliability in unstable network conditions

## GUI SERVER ENHANCEMENTS

- [ ] **[P2] Server Schedule Configuration Interface**
  - Visual cron expression builder with preview
  - Schedule enable/disable toggles
  - Next run time display
  - Schedule testing with dry-run capability
  - **Impact**: Makes scheduling accessible to non-technical users

- [ ] **[P2] Server Health Status Dashboard**
  - Real-time server status indicators (green/yellow/red)
  - Connection status for API and database
  - Active job count and queue depth
  - Recent error display with details
  - **Impact**: Provides instant visibility into system health

## CLI SERVER COMMANDS

- [ ] **[P2] Server Schedule Management**
  - `server schedule list` - Show all configured schedules
  - `server schedule add` - Add new cron schedule
  - `server schedule remove` - Remove existing schedule
  - `server schedule test` - Test cron expression
  - **Impact**: Full schedule control from command line

- [ ] **[P2] Backup & Restore Commands**
  - `server backup create` - Create manual backup
  - `server backup list` - List available backups
  - `server backup restore` - Restore from backup
  - `server backup auto` - Configure automatic backups
  - **Impact**: Critical data protection via CLI

### P3: MEDIUM (Scalability & Architecture)

- [ ] **[P3] Implement Repository Pattern**
  - Create AccountRepository, CheckinRepository interfaces
  - Separate database access from business logic
  - Enable easier testing with mock repositories
  - **Impact**: Better separation of concerns and testability

- [ ] **[P3] Add Caching Layer**
  - Implement LRU cache for frequently accessed accounts
  - Cache API responses for short periods (5-10 minutes)
  - Add cache invalidation on data updates
  - **Impact**: Reduces API calls and improves response times

- [ ] **[P3] Implement Circuit Breaker Pattern**
  - Add circuit breaker for API calls to prevent cascading failures
  - Configure failure thresholds and recovery times
  - Provide fallback mechanisms for critical operations
  - **Impact**: Improves system resilience

- [ ] **[P3] Error Handling Standardization**
  - Fix ignored errors in API client (body, _ := io.ReadAll patterns)
  - Implement consistent error wrapping with context
  - Add structured logging with error correlation IDs
  - **Impact**: Better debugging and error tracking

- [ ] **[P3] GUI Omnibox Enhancements**
  - Implement Omnibox for Check-ins and Routes
  - Check-ins: Search by AccountID and full_name
  - Routes: Search by RouteID and route_name
  - **Impact**: Improves user experience and feature completeness

- [ ] **[P3] GUI Explorer View Improvements**
  - Implement pagination for the data table
  - Add ability to sort table by clicking column headers
  - **Impact**: Better performance with large datasets

- [ ] **[P3] CLI View Pending Changes**
  - Add `push list` command to CLI to view pending changes
  - Mirror functionality from GUI Push tab
  - **Impact**: Improves CLI feature parity with GUI

## WEBHOOK & ACTION MANAGEMENT

- [ ] **[P3] Webhook Action Chaining**
  - Support multiple actions per webhook event
  - Conditional action execution based on webhook payload
  - Action dependencies and ordering
  - Parallel vs sequential execution options
  - **Impact**: Enables complex automation workflows

- [ ] **[P3] Rate Limiting & Throttling**
  - Implement per-endpoint rate limiting
  - Configurable rate limits by API key or IP
  - Sliding window or token bucket algorithms
  - Rate limit headers in responses
  - **Impact**: Prevents API abuse and ensures fair usage

- [ ] **[P3] Notification System**
  - Email notifications for job completion/failure
  - Webhook callbacks for external systems
  - Slack/Teams integration for alerts
  - Configurable notification rules and templates
  - **Impact**: Keeps stakeholders informed of system status

- [ ] **[P3] Job Queue & Priority System**
  - Implement job queue with priority levels
  - Concurrent job execution with worker pools
  - Job status tracking and history
  - Queue persistence across restarts
  - **Impact**: Efficient resource utilization and job management

- [ ] **[P3] Data Transformation Pipeline**
  - Field mapping and transformation rules
  - Data type conversion and validation
  - Custom transformation functions
  - Transformation preview and testing
  - **Impact**: Handles data format differences between systems

- [ ] **[P3] Database Connection Pooling**
  - Implement connection pool with configurable size
  - Connection health checks and auto-reconnect
  - Pool metrics and monitoring
  - Per-operation timeout configuration
  - **Impact**: Improves database performance and reliability

## GUI ADVANCED FEATURES

- [ ] **[P3] Job Monitoring Dashboard**
  - Real-time job status updates
  - Job history with filtering and search
  - Performance metrics per job type
  - Cancel/retry job operations
  - **Impact**: Complete visibility into background operations

- [ ] **[P3] Webhook Testing & Debug Tools**
  - Webhook payload inspector
  - Mock webhook generator
  - Webhook replay functionality
  - Response time analysis
  - **Impact**: Simplifies webhook integration development

## CLI ADVANCED COMMANDS

- [ ] **[P3] Job Management Commands**
  - `server jobs list` - Show running and queued jobs
  - `server jobs status <id>` - Detailed job status
  - `server jobs cancel <id>` - Cancel running job
  - `server jobs history` - View job history
  - **Impact**: Full job control from CLI

- [ ] **[P3] Action Testing Command**
  - `server test-action <config>` - Dry-run action configuration
  - Preview action effects without execution
  - Validate action configuration syntax
  - Performance profiling for actions
  - **Impact**: Safe action development and testing

### P4: LOW (Code Quality & Technical Debt)

- [ ] **[P4] Function Parameter Optimization**
  - Reduce HandleTestDBConnection from 7 to 3 parameters using config struct
  - Create parameter objects for complex method signatures
  - **Impact**: Improves API usability and reduces errors

- [ ] **[P4] Dependency Management**
  - Review 215 dependencies and remove unused ones
  - Consolidate similar dependencies where possible
  - Update to latest stable versions
  - **Impact**: Reduces build time and security vulnerabilities

### Performance Metrics to Track Post-Implementation

- [ ] **[P4] Set up monitoring for:**
  - API response times (target: <500ms p95)
  - Database query times (target: <100ms p95)
  - Memory usage (target: <100MB for typical operations)
  - Test execution time (target: <30 seconds full suite)
  - Code coverage (target: >80% for critical packages)

### Technical Debt Elimination

- [ ] **[P4] Replace interface{} with any (Go 1.18+)**
  - Update 15+ instances of interface{} usage
  - **Impact**: Better type safety and modern Go practices

- [ ] **[P4] Fix deprecated string.Title usage**
  - Replace with golang.org/x/text/cases
  - **Impact**: Future-proof codebase

- [ ] **[P4] GUI Feature Parity**
  - Add "Test" tab or button to GUI for API/database connection tests
  - Display application version in GUI (Home or About tab)
  - Add ability to create new event actions from scratch in Actions tab
  - Add feature to view webhook log and replay specific webhooks
  - **Impact**: Improves GUI completeness and user experience

- [ ] **[P4] Omnibox Configuration**
  - Add configuration button (4 dots) next to each Omnibox
  - Configuration should contain checkboxes to enable/disable specific search fields
  - All fields should be enabled by default
  - **Note**: Blocked until API supports field-specific filtering

- [ ] **[P4] Testing & CI/CD**
  - Add tests for saving configuration settings
  - Set up CI/CD pipeline (GitHub Actions) for automated testing and builds
  - **Impact**: Improves development workflow and code quality

- [ ] **[P4] Code Quality Improvements**
  - Update to Go 1.18+ features where beneficial
  - Review and optimize import statements
  - Standardize error messages and logging format
  - **Impact**: Maintains code quality and modern practices

## ADVANCED SERVER FEATURES

- [ ] **[P4] Visual Workflow Editor**
  - Drag-and-drop action builder in GUI
  - Visual representation of action chains
  - Conditional logic builder
  - Workflow templates and sharing
  - **Impact**: Makes complex automation accessible to all users

- [ ] **[P4] Performance Metrics Command**
  - `server metrics` - Display performance statistics
  - API call latency percentiles
  - Database query performance
  - Resource usage trends
  - **Impact**: Enables performance optimization

- [ ] **[P4] Multi-Endpoint Load Balancing**
  - Support multiple BadgerMaps API endpoints
  - Automatic failover on endpoint failure
  - Round-robin or weighted load distribution
  - Endpoint health monitoring
  - **Impact**: Improves reliability and scalability

- [ ] **[P4] Custom Action Scripts**
  - Support for Python/JavaScript action scripts
  - Sandboxed execution environment
  - Script version control and rollback
  - Shared script library
  - **Impact**: Enables custom business logic without code changes

---

## COMPLETED ITEMS (For Reference)

### Core Architecture Refactoring
- [x] **Refactor the Event System**: Overhauled event system to be more scalable and structured
- [x] **Use String-Based Event Types**: Replaced EventType integer enum with hierarchical string format
- [x] **Introduce Structured Event Payloads**: Defined specific structs for each event's payload
- [x] **Enhance Dispatcher with Wildcard Subscriptions**: Added pattern subscription support

### Server Customization (Original)
- [x] **Implement cron job support**: Added scheduling for API calls and actions
- [x] **Health Check Endpoint**: Added `/health` endpoint for status verification
- [x] **Webhook History and Replay**: Complete logging and replay functionality

### Server Security & Reliability (New - Completed Dec 2024)
- [x] **[P1] Webhook Signature Verification**: HMAC-SHA256 verification with replay attack prevention (`app/server/webhook_security.go`)
- [x] **[P1] Audit Logging System**: Comprehensive audit trail with rotation and async logging (`app/audit/audit_log.go`)
- [x] **[P2] Scheduled Sync Functionality**: Cron-based scheduling with timezone support and retry logic (`app/server/scheduler.go`)
- [x] **[P2] Data Validation & Health Checks**: Pre-sync validation and component health monitoring (`app/server/health_check.go`)
- [x] **[P2] Server Monitoring & Metrics**: Real-time metrics with percentile tracking and HTTP endpoint (`app/server/metrics.go`)
- [x] **[P2] Retry Mechanism with Exponential Backoff**: Integrated into scheduler for reliable job execution

### GUI Enhancements
- [x] **Implement Omnibox for Accounts**: Search by AccountID and full_name

### Logging
- [x] **File Logging**: Configurable log output to file with PWD default

---

**PROGRESS UPDATE (Dec 2024)**:
- **P1 Completed**: 2 of 5 items (40%) - Webhook security and audit logging ✅
- **P2 Completed**: 5 of 16 items (31%) - Core server automation foundation ✅
- **P3 Remaining**: 18 items - GUI/CLI enhancements and advanced features
- **P4 Remaining**: 12 items - Nice-to-have enhancements

**REMAINING ESTIMATED EFFORT**: 
- **P1 (Critical)**: 1-2 weeks - Test coverage, code duplication, conflict resolution
- **P2 (High)**: 4-5 weeks - Backup/restore, GUI/CLI tools, remaining automation
- **P3 (Medium)**: 4-6 weeks - Advanced features and management tools  
- **P4 (Low)**: 3-4 weeks - Nice-to-have enhancements
- **Total Remaining**: ~12-17 weeks to complete full server transformation
