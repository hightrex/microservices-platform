# Service Integration & Event-Driven Workflows

> **Purpose**: Document all service interactions, event flows, and cross-service workflows
> **Scope**: Phase 1 (Auth, Org, Gateway) + Phase 2 (Notification, Billing, File, Audit) integration

---

## Service Event Catalog

### Events Published by Each Service

#### Auth Service (Phase 1)
```yaml
Events Published:
  - user.created
      fields: user_id, email
      triggers: Registration, admin user creation
      status: IMPLEMENTED
      
  - user.updated
      fields: user_id, fields (map of updated fields)
      triggers: Profile update
      status: IMPLEMENTED
      
  - user.deleted
      fields: user_id
      triggers: User deactivation/deletion
      status: IMPLEMENTED
      
  - user.login
      fields: user_id, ip
      triggers: Successful authentication
      status: IMPLEMENTED
      
  - user.logout
      fields: session_id
      triggers: Logout action
      status: IMPLEMENTED
      
  - user.role_changed
      fields: user_id, new_role
      triggers: Role assignment/modification
      status: IMPLEMENTED
      
  - auth.failed
      fields: user_id, reason, ip
      triggers: Failed login attempt
      status: IMPLEMENTED
      
  - auth.password_changed
      fields: user_id
      triggers: Password change
      status: IMPLEMENTED
      
  - auth.mfa_enabled
      fields: user_id
      triggers: MFA setup completion
      status: IMPLEMENTED

  # Planned for Phase 2:
  - auth.lockout
  - auth.mfa_disabled
  - auth.api_key_created
  - auth.api_key_revoked
```

#### Organization Service (Phase 1)
```yaml
Events Published:
  - org.created
      fields: org_id, name, slug, owner_id, plan
      triggers: Organization creation
      status: IMPLEMENTED
      
  - org.updated
      fields: org_id, fields (map of updated settings)
      triggers: Organization settings update
      status: IMPLEMENTED
      
  - org.deleted
      fields: org_id
      triggers: Organization deletion
      status: IMPLEMENTED

  # Planned for Phase 2:
  - org.member_invited
  - org.member_joined
  - org.member_removed
  - org.module_enabled
  - org.module_disabled
  - org.settings_changed
```

#### Notification Service (Phase 2)
```yaml
Events Published:
  - notification.sent
      fields: notification_id, tenant_id, user_id, channel, event_type, status
      triggers: Successful notification delivery
      
  - notification.failed
      fields: notification_id, tenant_id, user_id, channel, error, retry_count
      triggers: Delivery failure
      
  - notification.template_created
      fields: template_id, tenant_id, name, channel
      triggers: Template creation
      
  - notification.preference_updated
      fields: user_id, tenant_id, channel, event_type, enabled
      triggers: User preference change

Events Consumed:
  - ALL events from Auth, Org, Billing, File services
      purpose: Trigger notifications based on events
```

#### Billing Service (Phase 2)
```yaml
Events Published:
  - subscription.created
      fields: subscription_id, tenant_id, plan_id, stripe_subscription_id
      triggers: New subscription
      
  - subscription.updated
      fields: subscription_id, tenant_id, old_plan, new_plan, change_type
      triggers: Plan upgrade/downgrade
      
  - subscription.canceled
      fields: subscription_id, tenant_id, cancel_at, immediately
      triggers: Subscription cancellation
      
  - subscription.trial_ending
      fields: subscription_id, tenant_id, days_remaining
      triggers: Trial expiration warning (7 days, 3 days, 1 day)
      
  - invoice.created
      fields: invoice_id, tenant_id, amount_due, due_date
      triggers: Invoice generation
      
  - invoice.paid
      fields: invoice_id, tenant_id, payment_id, amount_paid
      triggers: Successful payment
      
  - invoice.failed
      fields: invoice_id, tenant_id, failure_reason, retry_count
      triggers: Payment failure
      
  - invoice.overdue
      fields: invoice_id, tenant_id, days_overdue
      triggers: Invoice past due date
      
  - payment.succeeded
      fields: payment_id, tenant_id, invoice_id, amount
      triggers: Successful payment
      
  - payment.failed
      fields: payment_id, tenant_id, invoice_id, reason
      triggers: Payment failure
      
  - usage.recorded
      fields: tenant_id, metric_name, quantity, subscription_id
      triggers: Usage data point
      
  - usage.quota_exceeded
      fields: tenant_id, metric_name, limit, current_usage
      triggers: Over plan limits

Events Consumed:
  - org.created → Create default subscription
  - file.uploaded → Record storage usage
  - user.created → Record user count for billing
  - (API calls tracked via Gateway metrics)
```

#### File Service (Phase 2)
```yaml
Events Published:
  - file.uploaded
      fields: file_id, tenant_id, user_id, filename, size_bytes, content_type
      triggers: File upload completion
      
  - file.deleted
      fields: file_id, tenant_id, user_id, filename, size_bytes
      triggers: File deletion
      
  - file.scan_completed
      fields: file_id, tenant_id, scan_status, scan_duration
      triggers: Virus scan completion
      
  - file.quarantined
      fields: file_id, tenant_id, user_id, virus_name
      triggers: Infected file detected
      
  - quota.exceeded
      fields: tenant_id, quota_bytes, used_bytes, attempted_size
      triggers: Storage quota limit reached
      
  - quota.warning
      fields: tenant_id, quota_bytes, used_bytes, percentage
      triggers: Storage usage threshold (80%, 90%)

Events Consumed:
  - org.deleted → Delete all organization files
  - user.deleted → Delete user's files or reassign
```

#### Audit Service (Phase 2)
```yaml
Events Published:
  - audit.export_completed
      fields: export_id, tenant_id, file_id, format, record_count
      triggers: Audit export ready
      
  - audit.logs_archived
      fields: tenant_id, archived_count, date_range
      triggers: Old logs archived
      
  - audit.chain_broken
      fields: tenant_id, last_valid_hash, broken_at
      triggers: Hash chain integrity violation (CRITICAL)

Events Consumed:
  - ALL events from ALL services
      purpose: Capture complete audit trail
```

---

## Complete Workflow Sequences

### Workflow 1: User Registration & Organization Setup

```
┌─────────────┐
│   User      │
│ Registers   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Auth Service                                                │
│ 1. Validate email/password                                  │
│ 2. Hash password                                            │
│ 3. Create user in database                                  │
│ 4. Assign default role (org_owner)                          │
│ 5. Generate JWT tokens                                      │
│ 6. Publish: user.created                                    │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┬────────────────────┐
       ▼                     ▼                    ▼                    ▼
┌─────────────┐    ┌──────────────────┐  ┌─────────────────┐  ┌──────────────┐
│ Org Service │    │Billing Service   │  │Notification Svc │  │ Audit Service│
│             │    │                  │  │                 │  │              │
│ 1. Listen:  │    │ 1. Listen:       │  │ 1. Listen:      │  │ 1. Listen:   │
│   user.     │    │   user.created   │  │   user.created  │  │   user.      │
│   created   │    │                  │  │                 │  │   created    │
│             │    │ 2. Check if first│  │ 2. Load welcome │  │              │
│ 2. Create   │    │   user in tenant │  │   template      │  │ 2. Record:   │
│   org auto  │    │                  │  │                 │  │   - event    │
│   (if from  │    │ 3. Create:       │  │ 3. Send welcome │  │   - actor    │
│   register) │    │   - Stripe       │  │   email         │  │   - tenant   │
│             │    │     customer     │  │                 │  │   - ip       │
│ 3. Set org  │    │   - Free plan    │  │ 4. Publish:     │  │   - outcome  │
│   as tenant │    │     subscription │  │   notification. │  │              │
│             │    │   - Trial period │  │   sent          │  │ 3. Hash chain│
│ 4. Enable   │    │     (14 days)    │  │                 │  │              │
│   default   │    │                  │  │                 │  │ 4. Store in  │
│   modules   │    │ 4. Publish:      │  │                 │  │   append-only│
│             │    │   subscription.  │  │                 │  │   table      │
│ 5. Publish: │    │   created        │  │                 │  │              │
│   org.      │    │                  │  │                 │  │              │
│   created   │    │ 5. Publish:      │  │                 │  │              │
│             │    │   usage.recorded │  │                 │  │              │
└─────────────┘    └──────────────────┘  └─────────────────┘  └──────────────┘
       │                    │                      │                   │
       └────────────────────┴──────────────────────┴───────────────────┘
                                    │
                                    ▼
                          ┌─────────────────┐
                          │   User gets:    │
                          │ - Account       │
                          │ - Organization  │
                          │ - Free trial    │
                          │ - Welcome email │
                          │ - Audit logged  │
                          └─────────────────┘
```

**Event Flow:**
1. `user.created` → Triggers 4 parallel actions
2. `org.created` → Recorded by Audit, triggers Billing check
3. `subscription.created` → Notification sent, recorded by Audit
4. `notification.sent` → Recorded by Audit

**Total Events:** 4-5 events for one user registration

---

### Workflow 2: User Login

```
┌─────────────┐
│   User      │
│   Logs In   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Auth Service                                                │
│ 1. Validate credentials                                     │
│ 2. Check account lockout status                             │
│ 3. Verify MFA (if enabled)                                  │
│ 4. Generate JWT + refresh token                             │
│ 5. Create session record                                    │
│ 6. Update last_login_at                                     │
│ 7. Reset failed_login_count                                 │
│ 8. Log security event                                       │
│ 9. Publish: user.login                                      │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┐
       ▼                     ▼                    ▼
┌─────────────┐    ┌──────────────────┐  ┌──────────────┐
│ Audit Svc   │    │Notification Svc  │  │Billing Svc   │
│             │    │                  │  │              │
│ Record:     │    │ (Optional)       │  │ Record API   │
│ - Login     │    │ Send login       │  │ call for     │
│   success   │    │ notification if  │  │ metered      │
│ - IP/UA     │    │ from new device  │  │ billing      │
│ - Location  │    │ or location      │  │              │
└─────────────┘    └──────────────────┘  └──────────────┘

If login fails:
┌─────────────────────────────────────────────────────────────┐
│ Auth Service                                                │
│ 1. Increment failed_login_count                             │
│ 2. Check if should lock account                             │
│ 3. Log security event (FAILED_LOGIN)                        │
│ 4. Publish: auth.failed                                     │
│ 5. If locked: Publish auth.lockout                          │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┐
       ▼                     ▼                    ▼
┌─────────────┐    ┌──────────────────┐  ┌──────────────┐
│ Audit Svc   │    │Notification Svc  │  │(Other Svcs)  │
│             │    │                  │  │              │
│ Record:     │    │ Send security    │  │ May trigger  │
│ - Failed    │    │ alert if:        │  │ additional   │
│   login     │    │ - 5+ failures    │  │ monitoring   │
│ - Reason    │    │ - New IP         │  │              │
│ - IP        │    │ - Account locked │  │              │
└─────────────┘    └──────────────────┘  └──────────────┘
```

**Event Flow:**
- Success: `user.login` → Audit records, optional notification
- Failure: `auth.failed` → Audit records, notification on threshold
- Lockout: `auth.lockout` → Audit records, notification sent

---

### Workflow 3: File Upload with Full Processing

```
┌─────────────┐
│    User     │
│  Uploads    │
│    File     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ API Gateway                                                 │
│ 1. Validate JWT                                             │
│ 2. Check module_enabled (files)                             │
│ 3. Rate limit check                                         │
│ 4. Proxy to File Service                                    │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ File Service                                                │
│ 1. Validate file type (magic bytes)                         │
│ 2. Check storage quota                                      │
│ 3. Generate S3 key (tenant_id/uuid/filename)                │
│ 4. Stream upload to MinIO                                   │
│ 5. Calculate SHA-256 checksum                               │
│ 6. Save metadata (status=uploading)                         │
│ 7. Queue virus scan job                                     │
│ 8. Update metadata (status=available)                       │
│ 9. Publish: file.uploaded                                   │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┬───────────────────┐
       ▼                     ▼                    ▼                   ▼
┌─────────────┐    ┌──────────────────┐  ┌─────────────────┐  ┌──────────────┐
│ File Svc    │    │Billing Service   │  │Notification Svc │  │ Audit Service│
│ Background  │    │                  │  │                 │  │              │
│ Worker      │    │ 1. Listen:       │  │ 1. Listen:      │  │ 1. Record    │
│             │    │   file.uploaded  │  │   file.uploaded │  │   upload     │
│ 1. Download │    │                  │  │                 │  │   event      │
│   from S3   │    │ 2. Record usage: │  │ 2. Check if     │  │              │
│             │    │   - storage_gb   │  │   user wants    │  │ 2. Log:      │
│ 2. Scan via │    │   - file_count   │  │   upload        │  │   - file_id  │
│   ClamAV    │    │                  │  │   notifications │  │   - size     │
│             │    │ 3. Check quota:  │  │                 │  │   - user     │
│ 3. Parse    │    │   If exceeded:   │  │ 3. (Optional)   │  │   - ip       │
│   result    │    │   Publish:       │  │   Send upload   │  │              │
│             │    │   usage.quota_   │  │   confirmation  │  │              │
│ 4. Update   │    │   exceeded       │  │                 │  │              │
│   scan      │    │                  │  │                 │  │              │
│   status    │    │ 4. Publish:      │  │                 │  │              │
│             │    │   usage.recorded │  │                 │  │              │
└──────┬──────┘    └──────────────────┘  └─────────────────┘  └──────────────┘
       │
       ▼
  If CLEAN:                              If INFECTED:
┌─────────────┐                        ┌─────────────────────┐
│ 5. status=  │                        │ 5. status=quarantined│
│   available │                        │ 6. Publish:         │
│             │                        │   file.quarantined  │
│ 6. Generate │                        └──────┬──────────────┘
│   thumbnail │                               │
│   (if image)│                               ▼
│             │                        ┌─────────────────────┐
│ 7. Publish: │                        │ Notification Service│
│   file.scan_│                        │                     │
│   completed │                        │ Send CRITICAL alert │
│             │                        │ to org admins:      │
└─────────────┘                        │ "Virus detected!"   │
                                       └─────────────────────┘
```

**Event Flow:**
1. `file.uploaded` → Triggers Billing, Notification, Audit
2. `file.scan_completed` → Recorded by Audit
3. `usage.recorded` → Recorded by Audit
4. If infected: `file.quarantined` → CRITICAL alert sent

**Total Events:** 3-5 events per file upload

---

### Workflow 4: Subscription Upgrade/Downgrade

```
┌─────────────┐
│    User     │
│  Changes    │
│    Plan     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│ Billing Service                                             │
│ 1. Validate new plan exists                                 │
│ 2. Check current subscription                               │
│ 3. Calculate proration amount                               │
│ 4. Generate preview invoice                                 │
│ 5. Update Stripe subscription                               │
│ 6. Sync status locally                                      │
│ 7. Publish: subscription.updated                            │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┬───────────────────┐
       ▼                     ▼                    ▼                   ▼
┌─────────────┐    ┌──────────────────┐  ┌─────────────────┐  ┌──────────────┐
│ Org Service │    │Notification Svc  │  │ Audit Service   │  │Gateway/Modules│
│             │    │                  │  │                 │  │              │
│ 1. Listen:  │    │ 1. Listen:       │  │ 1. Record plan  │  │ 1. Update    │
│   subs.     │    │   subscription.  │  │   change        │  │   quota/rate │
│   updated   │    │   updated        │  │                 │  │   limits     │
│             │    │                  │  │ 2. Log:         │  │              │
│ 2. Update   │    │ 2. Send email:   │  │   - old_plan    │  │ 2. Refresh   │
│   module    │    │   "Plan changed  │  │   - new_plan    │  │   module     │
│   access    │    │   to X"          │  │   - proration   │  │   cache      │
│   based on  │    │                  │  │                 │  │              │
│   new plan  │    │ 3. Include:      │  │                 │  │              │
│             │    │   - New features │  │                 │  │              │
│ 3. If       │    │   - Quota limits │  │                 │  │              │
│   downgrade:│    │   - Billing info │  │                 │  │              │
│   Disable   │    │                  │  │                 │  │              │
│   premium   │    │                  │  │                 │  │              │
│   modules   │    │                  │  │                 │  │              │
│             │    │                  │  │                 │  │              │
│ 4. Publish: │    │                  │  │                 │  │              │
│   org.      │    │                  │  │                 │  │              │
│   module_   │    │                  │  │                 │  │              │
│   disabled  │    │                  │  │                 │  │              │
└─────────────┘    └──────────────────┘  └─────────────────┘  └──────────────┘
```

**Event Flow:**
1. `subscription.updated` → Org adjusts modules, Notification sent, Audit recorded
2. `org.module_disabled` (if downgrade) → Audit recorded, Gateway cache refreshed
3. Gateway enforces new limits immediately

---

### Workflow 5: Payment Success/Failure

```
Stripe Webhook → Billing Service

SUCCESS:
┌─────────────────────────────────────────────────────────────┐
│ Billing Service - Webhook Handler                          │
│ 1. Verify webhook signature                                 │
│ 2. Parse event: invoice.payment_succeeded                   │
│ 3. Find invoice by stripe_invoice_id                        │
│ 4. Update invoice status → paid                             │
│ 5. Record payment                                           │
│ 6. Publish: invoice.paid                                    │
│ 7. Publish: payment.succeeded                               │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┐
       ▼                     ▼                    ▼
┌─────────────┐    ┌──────────────────┐  ┌──────────────┐
│Notification │    │ File Service     │  │ Audit Service│
│   Service   │    │                  │  │              │
│             │    │ 1. Generate      │  │ Record all   │
│ Send:       │    │   invoice PDF    │  │ payment      │
│ - Receipt   │    │                  │  │ events       │
│ - Thank you │    │ 2. Store in S3   │  │              │
│             │    │                  │  │              │
│ Attach PDF  │    │ 3. Return file_id│  │              │
└─────────────┘    └──────────────────┘  └──────────────┘

FAILURE:
┌─────────────────────────────────────────────────────────────┐
│ Billing Service - Webhook Handler                          │
│ 1. Parse event: invoice.payment_failed                      │
│ 2. Update invoice status → failed                           │
│ 3. Increment retry_count                                    │
│ 4. Publish: invoice.failed                                  │
│ 5. Publish: payment.failed                                  │
└──────┬──────────────────────────────────────────────────────┘
       │
       ├─────────────────────┬────────────────────┐
       ▼                     ▼                    ▼
┌─────────────┐    ┌──────────────────┐  ┌──────────────┐
│Notification │    │ Org Service      │  │ Audit Service│
│   Service   │    │                  │  │              │
│             │    │ If 3+ failures:  │  │ Record       │
│ Send:       │    │                  │  │ payment      │
│ - Payment   │    │ 1. Disable       │  │ failure      │
│   failed    │    │   premium        │  │              │
│   alert     │    │   modules        │  │              │
│             │    │                  │  │              │
│ - Include:  │    │ 2. Grace period  │  │              │
│   - Reason  │    │   (7 days)       │  │              │
│   - Retry   │    │                  │  │              │
│     link    │    │ 3. Publish:      │  │              │
│   - Support │    │   org.module_    │  │              │
│             │    │   disabled       │  │              │
└─────────────┘    └──────────────────┘  └──────────────┘
```

**Event Flow:**
- Success: `invoice.paid` + `payment.succeeded` → Receipt sent, PDF generated
- Failure: `invoice.failed` + `payment.failed` → Alert sent, potential service degradation

---

## Service Dependency Matrix

| Service → Depends On | Auth | Org | Gateway | Notification | Billing | File | Audit |
|---------------------|------|-----|---------|--------------|---------|------|-------|
| **Auth**           | -    | No  | No      | No           | No      | No   | No    |
| **Org**            | Yes* | -   | No      | No           | No      | No   | No    |
| **Gateway**        | Yes  | Yes | -       | No           | No      | No   | No    |
| **Notification**   | No   | No  | No      | -            | No      | No   | No    |
| **Billing**        | No   | Yes*| No      | No           | -       | Yes* | No    |
| **File**           | No   | No  | No      | No           | No      | -    | No    |
| **Audit**          | No   | No  | No      | No           | No      | No   | -     |

*Soft dependency (via events, not direct API calls)

**Key:**
- Hard dependency = Direct API calls required
- Soft dependency = Event-driven interaction
- No = Fully independent

---

## Event Consumer Summary

| Service | Consumes Events From | Purpose | Status |
|---------|---------------------|---------|--------|
| **Org Service** | user.created | Auto-create organization for first user | ⏳ Pending (Phase 2) |
| **Billing Service** | org.created | Create subscription for new organization | ⏳ Pending (Phase 2) |
| **Billing Service** | file.uploaded | Track storage usage | ⏳ Pending (Phase 2) |
| **Billing Service** | user.created | Track user count for billing | ⏳ Pending (Phase 2) |
| **Notification Service** | ALL EVENTS | Send notifications based on any event | ⏳ Pending (Phase 2) |
| **Audit Service** | ALL EVENTS | Capture complete audit trail | ⏳ Pending (Phase 2) |

---

## Data Flow Patterns

### Pattern 1: Synchronous API Calls
```
User → Gateway → Service → Database → Response
```
Used for: CRUD operations, immediate responses

### Pattern 2: Asynchronous Events
```
Service A → Redis Streams → Service B (consumer)
```
Used for: Notifications, audit logging, cross-service updates

### Pattern 3: Hybrid (Common)
```
User → Gateway → Service A → Database
                           ↓
                    Publish Event
                           ↓
            Service B, C, D (async consumers)
```
Used for: Most write operations (create, update, delete)

---

## Integration Points by Phase

### Phase 1 Integration (Auth + Org + Gateway)
```yaml
Direct API Calls:
  - Gateway → Auth: Token validation (Implemented)
  - Gateway → Org: Module gating check (Implemented)
  
Event Flows:
  - user.created → Published by Auth, but Org Service consumer NOT YET implemented.
    * Current workaround: Frontend or API Gateway orchestration creates Org immediately after User.
  - org.created → Published by Org.
  
Data Sharing:
  - JWT contains: user_id, tenant_id, org_id, roles
  - All services validate via Gateway or Middleware
```

### Phase 2 Integration (Add Notification, Billing, File, Audit)
```yaml
New Direct API Calls:
  - Billing → File: Generate invoice PDF
  - Notification → File: Attach files to emails (optional)
  
New Event Flows:
  - user.created → Notification (welcome email)
  - user.created → Billing (track user count)
  - org.created → Billing (create subscription)
  - file.uploaded → Billing (track storage)
  - ALL events → Audit (complete trail)
  
Enhanced JWT:
  - Add subscription tier to JWT claims
  - Gateway enforces tier-based rate limits
```

---

## Quota Enforcement Integration

```yaml
Storage Quotas (File Service):
  - Plan: Free (1 GB), Pro (50 GB), Enterprise (unlimited)
  - Check: Before each file upload
  - Enforcement: File Service queries Billing Service for tenant plan
  - Event: quota.exceeded → Notification sent, Billing notified
  
User Quotas (Auth Service):
  - Plan: Free (5 users), Pro (50 users), Enterprise (unlimited)
  - Check: Before user creation/invitation
  - Enforcement: Auth Service queries Billing Service
  - Event: User creation blocked with upgrade prompt
  
API Rate Limits (Gateway):
  - Plan: Free (100/min), Pro (1000/min), Enterprise (10000/min)
  - Check: Every request
  - Enforcement: Gateway checks Redis + JWT tier claim
  - Event: Rate limit exceeded → Usage tracked for billing
```

---

## Failure Scenarios & Recovery

### Scenario 1: Notification Service Down
```
Impact:
  - Users still get created (Auth works)
  - Emails not sent (queued in Redis Streams)
  
Recovery:
  - When Notification Service restarts
  - Consumer reads from last acknowledged message
  - Sends all queued notifications
  
Result: Zero data loss, delayed notifications
```

### Scenario 2: Billing Service Down
```
Impact:
  - New subscriptions cannot be created
  - Existing functionality continues (cached tier in JWT)
  - Usage events queue in Redis Streams
  
Recovery:
  - Service restarts, processes queued events
  - Syncs with Stripe
  - Reconciles usage data
  
Result: Temporary service degradation, full recovery
```

### Scenario 3: File Service Down
```
Impact:
  - File uploads fail
  - Existing files still downloadable (via S3 presigned URLs)
  - Virus scanning paused
  
Recovery:
  - Service restarts
  - Background worker processes pending scans
  
Result: Upload interruption, existing files unaffected
```

### Scenario 4: Audit Service Down
```
Impact:
  - Events still published to Redis Streams
  - No immediate impact on user-facing features
  - Audit logs temporarily unavailable
  
Recovery:
  - Service restarts, processes all queued events
  - Hash chain continues from last valid entry
  - Compliance maintained
  
Result: Audit log gap filled, integrity verified
```

---

## Testing Integration Points

### Integration Test Checklist
```
□ User registration triggers:
  ✓ Organization creation
  ✓ Subscription creation
  ✓ Welcome email
  ✓ Audit log entries (4 events)
  
□ File upload triggers:
  ✓ Virus scan
  ✓ Usage tracking
  ✓ Optional notification
  ✓ Audit log
  
□ Subscription change triggers:
  ✓ Module access update
  ✓ Email notification
  ✓ Quota adjustment
  ✓ Audit log
  
□ Payment success triggers:
  ✓ Invoice marked paid
  ✓ Receipt email with PDF
  ✓ Audit logs
  
□ Payment failure triggers:
  ✓ Alert email
  ✓ Grace period starts
  ✓ Module disable (after grace period)
  ✓ Audit logs
```

---

## Performance Considerations

### Event Processing
- **Redis Streams**: Sub-millisecond latency
- **Consumer Groups**: Parallel processing, at-least-once delivery
- **DLQ**: Failed events don't block stream

### Caching Strategy
```yaml
Gateway:
  - JWT validation results: 5 minutes
  - Module gating decisions: 5 minutes
  - Rate limit state: Real-time (Redis)
  
Billing Service:
  - Plan details: 1 hour
  - Subscription status: 5 minutes
  
File Service:
  - File metadata: 10 minutes
  - Quota status: Real-time
```

### Database Queries
- All queries include `tenant_id` in WHERE clause
- Indexes on `(tenant_id, *)` for fast filtering
- Pagination for all list endpoints

---

## Summary

**Total Events in System**: ~40 event types
**Event Producers**: 6 services (Auth, Org, Notification, Billing, File, Audit)
**Event Consumers**: 3 services (Notification, Billing, Audit)
**Integration Patterns**: Sync API calls + Async events
**Failure Recovery**: Event replay from Redis Streams

**Key Principle**: Services are loosely coupled via events, making the system resilient, scalable, and maintainable.