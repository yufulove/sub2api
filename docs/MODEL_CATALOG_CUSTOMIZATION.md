# Model Catalog Customization

This branch keeps the `fionalovs.com` model-list cleanup in a form that can be replayed onto future upstream updates.

## Goal

Reduce noisy or obsolete model choices in:

- `测试账号连接`
- `添加账号 -> 模型白名单 / 模型映射`

The intended behavior is:

- OpenAI account testing uses a curated GPT-5 family list.
- Account creation defaults to the current platform whitelist instead of a stale fallback.
- Gemini and Antigravity test dialogs keep their prioritized image-capable ordering.

## Files touched

- `frontend/src/components/admin/account/AccountTestModal.vue`
- `frontend/src/components/account/AccountTestModal.vue`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/composables/modelCatalog.ts`
- `frontend/src/composables/useModelWhitelist.ts`

## Replay checklist after upgrading from upstream

1. Reapply `modelCatalog.ts`.
2. Reapply the `normalizePlatformModelOptions` and `pickDefaultTestModel` usage in both account test dialogs.
3. Keep `CreateAccountModal.vue` initializing the whitelist from `getModelsByPlatform(form.platform)`.
4. Keep the curated OpenAI list and preset mappings in `useModelWhitelist.ts`.
5. Run `pnpm build` in `frontend`.
6. Verify the built assets do not include legacy OpenAI strings such as:
   - `gpt-3.5-turbo-1106`
   - `gpt-4o-realtime-preview`
   - `o1-preview`

## Deployment note

For `fionalovs.com`, deployment must update both:

- the backend binary
- the static frontend files served by nginx

Updating only the embedded frontend is not enough for the public site.
