# 持衡 ChiHeng Desktop Design System

Status: implementation contract. Every visual value in frontend code must map to this document. Product boundary: local-first portfolio and market analysis; never present ChiHeng as a broker, trading terminal, or promise of returns.

## 0. Research Log

- Embedded references: shortlisted Kraken (financial data density), Linear (desktop restraint), and Notion (editorial clarity) → selected `minimalist-skill` + Kraken as source material because ChiHeng needs calm, high-density financial surfaces with unmistakable interaction states. Kraken names, logo, proprietary fonts, and exact brand purple are not copied.
- Product evidence: the `UI_Design.md`, `PRD.md`, and the feature inventory were reviewed → retained the information architecture needs, not a dated appearance.
- Product architecture evidence: Wails blueprint reviewed → desktop-first fixed shell, stale-data disclosure, long-task progress, offline readiness, and six bounded API surfaces drive the component system.
- Lazyweb: skipped — team direction had already selected a concrete reference layer; external screen research would add no decision needed for this deliverable.
- Imagen drafts: skipped — image generation is unavailable. A deterministic SVG vector master is the reference contract and platform assets are mechanically derived from it.

## 1. Atmosphere & Identity

ChiHeng is a quiet instrument panel: precise enough for dense portfolio work, restrained enough to support deliberate decisions. The signature is the **balanced ledger** — a central measuring axis with mirrored valuation bars, expressed in the app mark and echoed only in compact data dividers and loading states. Surfaces feel cool, light, and layered; violet identifies action and selection, while gain/loss colors communicate facts only.

Brand name: `持衡` in Chinese contexts, `ChiHeng` in technical metadata, and `持衡 ChiHeng` only at onboarding/about scale. Tagline: `看清持仓，理性权衡。` The mark is not a currency sign and must never be combined with arrows, rockets, coins, or candlesticks. Minimum mark size is 20 px in UI and 16 px only for tray/favicon use. Clear space equals one quarter of mark width.

## 2. Color

### Palette

| Role | CSS token | Light | Dark | Usage |
|---|---|---:|---:|---|
| Canvas | `--surface-canvas` | `#F6F7FB` | `#101116` | Window background |
| Primary | `--surface-primary` | `#FFFFFF` | `#17181F` | Main panes, cards |
| Secondary | `--surface-secondary` | `#F0F1F7` | `#1E2029` | Sidebar, grouped controls |
| Elevated | `--surface-elevated` | `#FFFFFF` | `#252733` | Menus, popovers, dialogs |
| Inverse | `--surface-inverse` | `#171821` | `#F7F7FB` | Tooltip and high-contrast moments |
| Text primary | `--text-primary` | `#171821` | `#F3F3F7` | Body, values, titles |
| Text secondary | `--text-secondary` | `#626577` | `#AFB2C1` | Captions, helper text |
| Text tertiary | `--text-tertiary` | `#8A8D9D` | `#858899` | Placeholders, disabled labels |
| Border default | `--border-default` | `#DADCE5` | `#343642` | Inputs, stronger dividers |
| Border subtle | `--border-subtle` | `#E8E9EF` | `#292B35` | Card and row separation |
| Accent 700 | `--accent-strong` | `#4938B8` | `#9D92F2` | Active press, accessible text links |
| Accent 600 | `--accent-primary` | `#5B4BD1` | `#8F83ED` | Primary action, selection, focus |
| Accent 500 | `--accent-hover` | `#6D5EDF` | `#A79DF3` | Hovered action |
| Accent soft | `--accent-soft` | `#EEEAFE` | `#2B2748` | Selected row, subtle badge |
| Success | `--status-success` | `#087A55` | `#4BD2A0` | Positive return, confirmed result |
| Success soft | `--status-success-soft` | `#E2F4EC` | `#173B31` | Success badge background |
| Warning | `--status-warning` | `#A15C00` | `#F3B65D` | Stale quote, caution |
| Warning soft | `--status-warning-soft` | `#FFF1DA` | `#422F17` | Warning badge background |
| Error | `--status-error` | `#B42332` | `#FF7F8B` | Loss, failure, destructive action |
| Error soft | `--status-error-soft` | `#FCE8EA` | `#481E24` | Error badge background |
| Info | `--status-info` | `#2869A8` | `#72B6F1` | Neutral information |
| Scrim | `--scrim` | `rgba(18,19,26,.44)` | `rgba(0,0,0,.62)` | Modal backdrop |

Rules: accent is scarce and interactive. Positive/negative colors never stand alone: pair them with sign, label, or icon. Charts use accent, success, warning, error, info in that order and provide pattern/shape differentiation. `--focus-ring` is `0 0 0 3px color-mix(in srgb, var(--accent-primary) 28%, transparent)`. Never add raw colors in components; extend this table first. Dark theme follows OS by default and can be overridden in Settings.

## 3. Typography

Use operating-system fonts so Chinese financial data renders crisply without bundled-font latency.

- UI: `Inter Variable, "SF Pro Text", "Segoe UI Variable", "Noto Sans SC", "Microsoft YaHei UI", sans-serif`
- Numeric/mono: `"SFMono-Regular", "Cascadia Mono", "Roboto Mono", "Noto Sans Mono CJK SC", monospace`
- Tabular numbers: `font-variant-numeric: tabular-nums lining-nums`; decimals align by column, never by added spaces.

| Token | Size / line | Weight | Tracking | Use |
|---|---|---:|---:|---|
| `--type-display` | `32px / 40px` | 680 | `-0.025em` | Onboarding and empty-state headline only |
| `--type-title` | `24px / 32px` | 650 | `-0.018em` | Page title |
| `--type-heading` | `18px / 26px` | 620 | `-0.01em` | Panel heading |
| `--type-body` | `14px / 22px` | 400 | `0` | Default UI text |
| `--type-body-strong` | `14px / 22px` | 600 | `0` | Labels and emphasis |
| `--type-small` | `12px / 18px` | 450 | `0.005em` | Metadata and helper text |
| `--type-data-lg` | `28px / 34px` | 620 | `-0.02em` | Portfolio totals |
| `--type-data` | `14px / 20px` | 520 | `0` | Tables and quotes |
| `--type-caption` | `11px / 16px` | 600 | `0.035em` | Compact status labels; uppercase only in English |

Body text never drops below 12 px. Chinese text is not letter-spaced. Long titles clamp rather than force a wider shell. Amounts include units in adjacent secondary text; never encode meaning through precision alone.

## 4. Spacing & Layout

Base unit is 4 px. Tokens: `--space-1:4px`, `--space-2:8px`, `--space-3:12px`, `--space-4:16px`, `--space-5:20px`, `--space-6:24px`, `--space-8:32px`, `--space-10:40px`, `--space-12:48px`, `--space-16:64px`.

The desktop shell uses titlebar `48px`, sidebar `224px` (compact `72px`), contextual inspector `320px` when present, and content max `1440px`. Page padding is 24 px at ≥1024, 20 px at 768–1023, 16 px below 768. Dense tables own their horizontal scroll; primary page content never does. Every flex/grid scroll child sets `min-inline-size:0` and `min-block-size:0`. At widths below 768 the inspector becomes a modal sheet and the sidebar becomes an overlay; at 375 all forms become one column.

Radius tokens: `--radius-sm:6px`, `--radius-md:10px`, `--radius-lg:14px`, `--radius-round:999px`. Buttons never use pill radius; badges may. Minimum target is 32×32 desktop, 44×44 when touch input is detected. Data density modes adjust row height only: compact 36 px, comfortable 44 px.

## 5. Components

All primitives require default, hover, active, focus-visible, disabled, loading, error where applicable. `data-state` drives styling; icons never carry state alone.

### App shell and navigation

- **Structure**: draggable titlebar; non-draggable window controls/actions; sidebar navigation; one scroll-owning `main`; optional inspector.
- **States**: nav default/hover/current/focus/disabled; collapsed sidebar retains tooltips and accessible labels.
- **Accessibility**: landmarks, skip-to-content, current page via `aria-current="page"`; drag regions never cover controls.
- **Motion**: inspector/sidebar opacity + transform, 180 ms; no width animation.

### Button and icon button

- **Variants**: primary, secondary, ghost, danger; sizes 32/36/40. Primary uses accent; only one primary action per region.
- **Anatomy**: label plus optional leading Phosphor icon at 16 px, 1.75 px stroke; icon-only controls require tooltip and accessible name.
- **States**: hover changes tone, active translates `scale(.98)`, focus uses `--focus-ring`, loading preserves width, disabled is non-interactive and visually muted.

### Field controls

- **Structure**: persistent label, control, optional prefix/suffix, helper or error line. Includes text, numeric, select, date, search, segmented control, checkbox, switch.
- **States**: empty, filled, hover, focus, invalid, disabled, readonly, loading. Error text identifies the fix; placeholders are examples, not labels.
- **Numeric rules**: accept locale input but normalize visibly; destructive/financial submissions show a review step.

### Data table and asset row

- **Structure**: semantic table, sticky header inside table scroll owner, sortable column buttons, row actions revealed on focus as well as hover.
- **States**: loading skeleton, empty, error/retry, selected, stale, partially available. Stale values retain value but show timestamp + warning label.
- **Layout**: first column sticky only when it does not obscure horizontal context. Right-align numeric cells with tabular figures.

### Metric card and chart panel

- **Structure**: label, primary value, comparison, freshness/source, optional compact chart. Cards are not clickable unless the whole surface has a declared destination.
- **States**: loading, no-data, stale, error, privacy-masked. Chart tooltip is keyboard reachable; summary table is available to assistive technology.

### Status badge

- **Variants**: neutral, info, success, warning, error, running. Structure is icon/shape + text; 6 px radius, never decorative pills.
- **Copy**: use explicit states such as `数据已过期`, `离线可用`, `分析中`; never a lone colored dot.

### Dialog, sheet, popover, tooltip

- **Dialog**: confirm irreversible actions and multi-step financial changes; focus trapped, Escape closes except during atomic commit, return focus on close.
- **Sheet**: contextual detail below 768. Popover handles lightweight selection; tooltip never holds required information.
- **Motion**: scrim opacity and surface transform only, 180–220 ms.

### Toast, inline notice, progress

- Toast reports transient completion; inline notice reports persistent or actionable state. Errors stay until dismissed/resolved. Task progress shows name, percentage when known, cancel affordance, and final result.
- `aria-live="polite"` for progress/success; assertive only when immediate action prevents data loss.

### Empty, loading, and error state

- Empty state names why it is empty and offers one next action. Skeletons mirror final geometry and stop after content loads. Error state includes plain-language cause, retry if safe, and request ID only in expandable details.

### Iconography

- Product UI uses `@phosphor-icons/react`, regular weight, 16/20 px; use bold only at 12–16 px when regular loses clarity. The local sprite `assets/brand/chiheng-ui-icons.svg` is reserved for bootstrap/static HTML where React icons are unavailable.
- Required mappings: overview=`squares-four`, holdings=`wallet`, market=`chart-line-up`, search=`magnifying-glass`, plans=`calendar-check`, diagnosis=`pulse`, settings=`gear-six`, refresh=`arrow-clockwise`, stale=`clock-warning`, offline=`cloud-slash`, success=`check-circle`, warning=`warning`, error=`x-circle`, info=`info`, import=`download-simple`, export=`upload-simple`, external=`arrow-square-out`.
- No emoji, text glyphs, national flags, or mixed icon libraries. Do not use the brand mark as a generic action icon.

## 6. Motion & Interaction

| Token | Value | Use |
|---|---|---|
| `--motion-micro` | `120ms cubic-bezier(.2,.8,.2,1)` | Press, check, hover feedback |
| `--motion-standard` | `180ms cubic-bezier(.2,.8,.2,1)` | Menu, tooltip, tab indicator |
| `--motion-emphasis` | `260ms cubic-bezier(.16,1,.3,1)` | Dialog/sheet entrance |

Animate only transform, opacity, and filter. Motion communicates relationship or state, never decoration. The balanced-ledger loader may fade alternating bars at 720 ms only while a real task is active. `prefers-reduced-motion: reduce` removes transforms and nonessential repetition; progress remains understandable from text. Hover is enhancement, never the sole reveal path. Keyboard order follows visual order and Enter/Space behavior follows native semantics.

## 7. Depth & Surface

Strategy: **mixed tonal shift with restrained elevation**. Persistent hierarchy uses color and 1 px borders; floating surfaces use shadow.

- `--shadow-subtle: 0 1px 2px rgba(19,20,29,.05)` — cards needing edge separation.
- `--shadow-float: 0 8px 24px rgba(19,20,29,.10), 0 2px 6px rgba(19,20,29,.06)` — menus/popovers.
- `--shadow-modal: 0 20px 56px rgba(10,11,17,.18), 0 4px 12px rgba(10,11,17,.08)` — dialogs only.

No glassmorphism, glow, noisy texture, or decorative gradient in product UI. The application icon alone uses a subtle multi-stop violet material to maintain depth on OS launch surfaces. Nested cards are discouraged; use headings/dividers within one surface.

## 8. Accessibility Constraints & Accepted Debt

Target WCAG 2.2 AA: 4.5:1 normal text, 3:1 large text and essential graphics, visible focus on every interactive control, full keyboard reachability, correct screen-reader names/states, 200% zoom without lost actions, and motion/contrast/color-scheme preferences respected. Chinese and English copy must remain understandable at 375 px and under text expansion. Charts expose a text summary/table. Gain/loss and stale/offline states always have redundant text or icon shape. Destructive investment actions require confirmation and never default focus on the destructive button.

Primary personas: (1) a detail-oriented long-term investor scanning dense numbers by keyboard; (2) a casual user who needs plain language and safe defaults; (3) a low-vision user using 200% zoom/high contrast; (4) a user with color-vision deficiency interpreting charts and returns; (5) an offline user relying on last-known data and freshness metadata.

| Accepted debt | Location | Reason | Owner / exit |
|---|---|---|---|
| Platform high-contrast assets require OS packaging QA | Windows/macOS/Linux packages | Source assets exist, but native shells are not yet packaged in this phase | Release owner; verify in signed/package candidates before v1 |
| Primitive browser showcase evidence pending | Frontend component harness | This deliverable establishes the contract before components exist | Frontend owner; must pass 375/768/1280 state QA before product screens |

Asset contract: vector masters in `assets/brand/`; web-ready copies in `frontend/public/brand/`; Wails/platform launch assets in `build/`. Never redraw or recolor masters. Regenerate raster assets from `assets/brand/chiheng-app-icon.svg`, retaining transparency and sRGB.
