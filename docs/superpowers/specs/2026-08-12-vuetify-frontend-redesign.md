# Frontend Redesign with Vuetify

**Date:** 2026-08-12
**Status:** Approved (post-brainstorming)
**Scope:** Frontend only. Backend API is unchanged.

## Background

The current frontend is functional but visually plain: hand-rolled CSS, no design system, a few inline-styled tables and inputs. It ships behind the same Go binary as the backend and is the only UI for both the public app catalog and the admin console.

The user requested a redesign using Vuetify to "beautify" the platform. Brainstorming produced a clear direction: **Industrial Blueprint** aesthetic — deep navy + cool teal palette, hairline borders, no shadows or rounded corners, monospace for data, sans for human text, both dark and light themes with a system-preference default.

The existing API contract and routing stay as-is. This spec covers only the visual layer: dependencies, theme, layout shell, and per-page treatment.

## Design Direction

Aesthetic: **Industrial Blueprint**. The platform reads as a live control panel, not a marketing site.

- Dark theme: deep navy `#0d1b2a` background, cool teal `#a8dadc` accent, hairline `#415a77` borders, light text `#e0e1dd`.
- Light theme: cream paper `#f4f1ea` background, navy ink, teal accent, same hairline borders.
- Signature element: a small filled `●` in the accent color followed by an uppercase mono label, used at every "live / active / enabled" state.
- No shadows. No border-radius. No gradients. Restraint is the design.

## Visual Tokens

### Dark "Control Room"

| Token | Hex | Use |
|---|---|---|
| `bg` | `#0d1b2a` | Page background |
| `surface` | `#1b263b` | Card, panel, table row |
| `surface-2` | `#243349` | Hover, selected |
| `border` | `#415a77` | Hairlines, dividers, focus rings |
| `text` | `#e0e1dd` | Primary text |
| `text-mute` | `#778da9` | Labels, captions |
| `accent` | `#a8dadc` | Links, primary action, live indicator |
| `success` | `#a8dadc` | Public, enabled |
| `warning` | `#f4a261` | Password, near-expiry |
| `danger` | `#e07a5f` | Taken-down, expired, destructive |

### Light "Blueprint Paper"

| Token | Hex | Use |
|---|---|---|
| `bg` | `#f4f1ea` | Page background |
| `surface` | `#e8e3d6` | Card, panel, table row |
| `surface-2` | `#dcd6c5` | Hover, selected |
| `border` | `#415a77` | Hairlines (navy ink) |
| `text` | `#0d1b2a` | Primary text |
| `text-mute` | `#415a77` | Labels, captions |
| `accent` | `#0d6e6e` | Links, primary action |
| `success` | `#0d6e6e` | Public, enabled |
| `warning` | `#b8651b` | Password, near-expiry |
| `danger` | `#a83232` | Taken-down, expired |

### Typography

- **Sans (display + body):** Inter, weights 400 / 500 / 600. Tracking `-0.02em` on display sizes.
- **Mono (data):** JetBrains Mono. Used for: version names, file sizes, sha256, timestamps, channel names, version codes, mono labels.
- **Type scale (rem):** 12 / 14 / 16 / 20 / 28 / 36. Headings use weight 500, not 700 — restraint.
- **Eyebrows:** all-caps, `letter-spacing: 0.15em`, `text-mute` color. Used above section titles and inline as labels.

### Spacing & layout

- 4px base. Standard gaps: 8 / 12 / 16 / 24 / 32.
- Max content width 1200px.
- Cards: 1px border, `border-radius: 0` globally, no shadows.
- Section dividers are full-width 1px hairlines, never blank space.

## Signature Element — the live dot

A `<StatusDot mode="public" />` component renders a `●` glyph in the mode's color followed by the mode label in mono uppercase. It is the recurring visual heartbeat of the platform — every place a state is shown uses it (app list, version rows, admin lists, status panels). The component encapsulates the signature so the look stays consistent.

## Layout & Navigation

### App shell

Single column with a 56px top bar. No left drawer — the platform has too few routes to justify one.

**Top bar contents:**
- Left: wordmark `▌ DISTRIBUTION` (mono, `accent`, all-caps)
- Center: route tabs — context-aware
- Right: theme toggle, then either Login link or user avatar with logout

**Tabs by state:**

| State | Tabs |
|---|---|
| Unauthenticated | Apps, Login |
| Authenticated | Apps, Admin, Upload |

**Responsive:** at `< sm` (600px), tabs collapse to a `v-menu` hamburger.

### Per-page layout

- **Home** (`/`): eyebrow + display title + 3-up stat strip + card grid. 3-up at desktop, 2-up at tablet, 1-up at mobile.
- **AppDetail** (`/app/:id`): two-column at `≥ md`. Left = app metadata + channel pills. Right = version list, hairline-divided rows.
- **Login** (`/login`): centered 360px card on a faint blueprint-grid background (subtle repeating SVG).
- **Admin** (`/admin`): page header + inline create form + data table.
- **AdminApp** (`/admin/app/:id`): tabs for `VERSIONS` · `CHANNELS` · `STATS`.
- **Upload** (`/admin/upload`): three sections (`FILE` / `METADATA` / `ACCESS`) separated by hairlines, no stepper.

## Components & Patterns

### Vuetify component map

| UI need | Component | Customization |
|---|---|---|
| Top bar | `v-app-bar` density=compact, flat, no shadow | 1px bottom border, mono wordmark |
| Tabs | `v-tabs` (slider hidden via CSS) | Active underline = 1px `accent` |
| Card / tile | `v-card` flat | 1px border, 0 radius, hover → `surface-2` |
| Data table | `v-data-table` density=comfortable | Hide header bg, header row = mono eyebrows, hairline row dividers |
| Form fields | `v-text-field`, `v-select`, `v-textarea` variant=outlined density=comfortable | 1px border, 0 radius, mono for numeric fields, label = mono uppercase |
| Buttons | `v-btn` variant=outlined (default) / flat (in-row) / `accent` (primary) | All caps, `letter-spacing: 0.1em`, 0 radius, 1px border |
| Dialogs | `v-dialog` with `v-card` | 1px border, no shadow, mono title in eyebrow style |
| Toasts | `v-snackbar` | Hairline border, mono message, 0 radius |
| File input | `v-file-input` variant=outlined | Custom dropzone with hairline dashed border |
| Theme toggle | `v-btn` icon | `mdi-theme-light-dark` / `mdi-weather-sunny` / `mdi-weather-night` |
| Status chips | `v-chip` variant=outlined size=small | Hairline border, mono text, color = mode |

### Custom components

- **`<StatusDot mode="public" />`** — the signature element. Renders `●` + uppercase mono label. Color resolves from theme tokens.
- **`<MonoText>`** — `<span>` with monospace font for inline data (hashes, sizes, versions).
- **`<StatBlock label="APPS" :value="3" />`** — bordered box for the home page stat strip. Label = mono uppercase eyebrow, value = large sans number.

### Icons

`@mdi/js` (already bundled with Vuetify). Icons only where they add clarity — theme toggle, action buttons, status dots. No decorative icons.

### Loading & empty states

- **Loading:** thin 1px `accent` progress bar at the top of the content area. No spinner.
- **Empty:** centered mono text with a small eyebrow above, e.g. `▌ NO APPS` / `no applications yet`.
- **Error:** hairline-bordered alert with mono title, body in sans, `accent` icon.

### Accessibility

- All interactive elements have visible 1px `accent` focus rings (offset, not the default browser blue).
- Color is never the only signal — text labels accompany all status indicators.
- `prefers-reduced-motion: reduce` disables transitions.

## Theming

### Three-state toggle

Cycles `system` → `light` → `dark` → `system`. Icons:
- `system`: `mdi-theme-light-dark`
- `light`: `mdi-weather-sunny`
- `dark`: `mdi-weather-night`

Tooltip shows the current state name on hover.

### Persistence & defaults

- Selected state stored in `localStorage.disapp-theme` as `'system' | 'light' | 'dark'`.
- On first visit (no stored value), read `window.matchMedia('(prefers-color-scheme: dark)')` and pick `dark` if true, `light` otherwise.
- `system` mode re-reads the media query on each page load.
- The chosen value is applied as a `data-theme="light|dark"` attribute on `<html>`, so both Vuetify and the custom CSS react to one source of truth.

## File Structure

New and changed files:

```
frontend/
├── package.json                     # + vuetify, @mdi/js, sass, vite-plugin-vuetify
├── src/
│   ├── main.ts                      # install Vuetify plugin
│   ├── plugins/
│   │   └── vuetify.ts               # createVuetify({ themes, defaults })
│   ├── composables/
│   │   └── useTheme.ts              # localStorage + media query sync
│   ├── components/
│   │   ├── StatusDot.vue            # signature element
│   │   ├── MonoText.vue             # inline mono span
│   │   ├── StatBlock.vue            # home page stat tile
│   │   └── AppShell.vue             # v-app-bar + tabs + theme toggle
│   ├── styles/
│   │   ├── tokens.css               # CSS custom props for both themes
│   │   ├── fonts.css                # @import Inter + JetBrains Mono from Google Fonts
│   │   └── overrides.scss           # global Vuetify overrides (radius, elevation, etc.)
│   ├── views/
│   │   ├── Home.vue                 # rewritten with Vuetify
│   │   ├── AppDetail.vue            # rewritten with Vuetify
│   │   ├── Login.vue                # rewritten with Vuetify
│   │   └── admin/
│   │       ├── Admin.vue
│   │       ├── AdminApp.vue
│   │       └── Upload.vue
│   └── router/index.ts              # unchanged
```

## API Compatibility

No backend changes. Existing endpoints serve the same shapes; the frontend only changes how data is rendered. `api/client.ts` and `api/types.ts` stay as-is.

## Testing

- `npm run build` produces a working single-file bundle.
- All six routes (`/`, `/app/:id`, `/login`, `/admin`, `/admin/app/:id`, `/admin/upload`) load and render real data from the existing API.
- Theme toggle cycles through three states and persists across reloads.
- `prefers-color-scheme: dark` triggers dark mode when state is `system`.
- A Playwright smoke test (already present in the plan) continues to pass.

## Out of scope

- Backend changes.
- New admin features.
- Mobile-app packaging or PWA.
- Internationalization (UI copy stays in the existing language; only structure changes).
- Animations beyond a single 100ms hover transition on cards and buttons.
