# Frontend UX Baseline

## Technology Foundation

| Concern | Technology | Source |
|---|---|---|
| Framework | Vue 3 (Composition API) | `frontend/package.json` |
| Build tool | Vite | `frontend/vite.config.js` |
| State management | Pinia | `frontend/src/stores/` |
| Routing | Vue Router 4 | `frontend/src/router/index.js` |
| HTTP client | Axios | `frontend/src/api/client.js` |
| Charts | ECharts | Used in `AnalyticsPage.vue` |
| Styling | SCSS with design tokens | Component `<style>` blocks |
| Testing | Vitest | `frontend/vitest.config.js` |

## Visual Hierarchy Rules

### Component Library

All shared UI components are in `frontend/src/components/common/`:

| Component | Purpose | Key Props |
|---|---|---|
| `AppButton.vue` | Action buttons | variant, size, disabled, loading |
| `AppInput.vue` | Text inputs | label, type, error, placeholder |
| `AppSelect.vue` | Dropdown selects | label, options, error |
| `AppTable.vue` | Data tables | columns, rows, sortable, paginated |
| `AppDialog.vue` | Modal dialogs | title, visible, onConfirm, onCancel |
| `AppChip.vue` | Status indicators | status, label, variant, size |
| `AppBreadcrumb.vue` | Navigation path | items (name, path pairs) |
| `AppFileUpload.vue` | File upload | accept, maxSize, onUpload |
| `AppToast.vue` | Notification toasts | message, type, duration |
| `AppLoadingState.vue` | Loading placeholders | message |
| `AppEmptyState.vue` | Empty state display | title, description, actionLabel |
| `AppErrorState.vue` | Error state display | message, onRetry |

### Page Layout

All pages follow a consistent structure:
1. **Page header**: Title + breadcrumb navigation
2. **Action bar**: Primary actions (create, import, filter)
3. **Content area**: Table, cards, or form
4. **Pagination**: Bottom-aligned for list views

### Pages

| Page | Route | File | Description |
|---|---|---|---|
| Login | `/login` | `LoginPage.vue` | Username/password form |
| Org Tree | `/org` | `OrgTreePage.vue` | Hierarchical org node management |
| Master Data | `/master/:entity` | `MasterDataPage.vue` | Entity-specific CRUD with tabs |
| Playback | `/playback` | `PlaybackPage.vue` | Media library with audio player |
| Analytics | `/analytics` | `AnalyticsPage.vue` | KPI dashboard with ECharts |
| Ingestion | `/ingestion` | `IngestionPage.vue` | Import sources and job monitoring |
| Reports | `/reports` | `ReportsPage.vue` | Schedule management and run history |
| Security Admin | `/security` | `SecurityAdminPage.vue` | Keys, fields, retention, holds |

## Interaction Feedback Requirements

### Loading States

Every data-fetching operation shows a loading indicator:
- `AppLoadingState` component for full-page loads
- Inline loading spinners for button actions (via `loading` prop on `AppButton`)

### Error Handling

- API errors display the `message` from the error response
- Network errors show "Unable to reach the server. Check your connection."
- `AppErrorState` component provides a retry button
- Toast notifications for transient errors (via `AppToast`)

### Form Validation

- Real-time validation feedback on `AppInput` components via `error` prop
- Entity-specific validation messages from the API (e.g., "SKU code must match pattern...")
- Disabled submit buttons until required fields are filled

### Optimistic vs. Pessimistic Updates

All mutations are pessimistic: the UI waits for the API response before updating state. This ensures data consistency given the multi-user, multi-org environment.

### Auto-Logout

The auth store (`frontend/src/stores/auth.js`) tracks user activity and automatically logs out after 30 minutes of inactivity:

```javascript
const IDLE_TIMEOUT_MS = 30 * 60 * 1000; // 30 minutes
```

Activity events tracked: `mousedown`, `keydown`, `scroll`, `touchstart`

The 401 response interceptor in `frontend/src/api/client.js` also triggers immediate logout if the server rejects a token.

## Status Chip Conventions

**Component**: `frontend/src/components/common/AppChip.vue`

### Status-to-Variant Mapping

| Status Value | Chip Variant | Color Scheme | Use Case |
|---|---|---|---|
| `active` | `success` | Green background, green text | Active records, healthy systems |
| `effective` | `success` | Green | Effective versions |
| `ready` | `success` | Green | Report runs ready for download |
| `inactive` | `neutral` | Gray | Deactivated records |
| `archived` | `neutral` | Gray | Archived versions |
| `draft` | `info` | Blue | Draft versions |
| `review` | `warning` | Yellow/amber | Versions under review |
| `pending` | `warning` | Yellow/amber | Pending approvals, pending jobs |
| `running` | `info` | Blue | Jobs in progress |
| `blocked` | `warning` | Yellow/amber | Jobs blocked by dependencies |
| `awaiting-ack` | `warning` | Yellow/amber | Failed jobs needing acknowledgment |
| `failed` | `danger` | Red | Failed operations |
| `error` | `danger` | Red | Error states |

Source: `frontend/src/components/common/AppChip.vue:32-46`

### Chip Structure

Each chip renders:
1. A colored dot indicator (`app-chip__dot`)
2. A text label (`app-chip__label`)

```html
<span class="app-chip app-chip--success">
  <span class="app-chip__dot" />
  <span class="app-chip__label">Active</span>
</span>
```

### Size Variants

| Size | Class | Padding | Font Size |
|---|---|---|---|
| `md` (default) | `.app-chip` | 3px 10px | `$font-size-xs` |
| `sm` | `.app-chip--sm` | 2px 8px | 10px |

### Color Token System

The chip component uses SCSS design tokens:

| Variant | Background Token | Text Token | Dot Token |
|---|---|---|---|
| success | `$color-success-50` | `$color-success-700` | `$color-success-500` |
| warning | `$color-warning-50` | `$color-warning-700` | `$color-warning-500` |
| danger | `$color-danger-50` | `$color-danger-700` | `$color-danger-500` |
| info | `$color-primary-50` | `$color-primary-700` | `$color-primary-500` |
| neutral | `$color-neutral-100` | `$color-neutral-600` | `$color-neutral-400` |

## API Communication Pattern

### Axios Client

**File**: `frontend/src/api/client.js`

Features:
- Base URL: `/api/v1` (configurable via `VITE_API_URL`)
- 30-second timeout
- Auto-attaches JWT from localStorage
- Generates correlation ID per request
- Auto-logout on 401 response
- Normalized error objects for consistent error handling

### Per-Domain API Modules

Each domain has a dedicated API module in `frontend/src/api/`:

| Module | File | Endpoints Covered |
|---|---|---|
| Auth | `auth.js` | login, logout, refresh |
| Org | `org.js` | tree, nodes, context |
| Master | `master.js` | records, history |
| Versions | `versions.js` | versions, items, diff |
| Ingestion | `ingestion.js` | sources, jobs, checkpoints, failures |
| Playback | `playback.js` | media, stream, lyrics |
| Analytics | `analytics.js` | KPIs, definitions, trends |
| Reports | `reports.js` | schedules, runs, download |
| Audit | `audit.js` | logs, delete requests |
| Security | `security.js` | fields, keys, retention, holds, purge |
| Integrations | `integrations.js` | endpoints, deliveries, connectors |

## Role-Based UI Rendering

The router enforces page-level access via `meta.roles`, and the auth store provides role-checking utilities:

```javascript
// Check single role
auth.hasRole('system_admin')

// Check if user has any of the specified roles
auth.hasAnyRole(['system_admin', 'operations_analyst'])
```

Components can conditionally render based on role:
```vue
<template>
  <AppButton v-if="auth.hasRole('system_admin')" @click="manageOrg">
    Manage Organisation
  </AppButton>
</template>
```
