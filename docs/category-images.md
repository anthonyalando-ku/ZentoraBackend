# Optional category images

This change adds category images only. It preserves the storefront category grid's layout, links, labels, and existing initials fallback; it does not redesign the homepage or dashboard.

## Deployment

Apply `internal/db/migrations/010_category_images.sql` once with the API's existing database role and search_path, then deploy backend followed by frontend. Use `psql -v ON_ERROR_STOP=1 -f internal/db/migrations/010_category_images.sql` with the normal secure connection setup. The migration owns its transaction, has a five-second lock timeout, and adds nullable `product_categories.image_url` without defaults or historical backfills. Do not replay initial schema scripts. Keep this additive column if rolling back application code.

No production migration or live ImageKit upload was performed during development.

## API contract

Existing endpoints and authentication remain unchanged:

- `POST /api/v1/admin/catalog/categories`
- `PUT /api/v1/admin/catalog/categories/:id`
- Existing category list, detail, tree, and product-category endpoints return `image_url` as a string or null alongside existing fields. Closure-only responses remain relationships, not category records.

Existing JSON requests still work. Create or change a URL with `image_url: "https://example.com/category.png"`. Omit `image_url` (or send null) on update to preserve the current image. Send `image_url: ""` to remove it. No image is required for create or update.

For uploads, use multipart/form-data with one `image` file and a `data` text field containing the same category JSON, e.g. `{"name":"Electronics"}`. Do not submit a nonempty image URL and file together. Metadata and the resulting image URL are saved in the same category operation; upload errors do not create or mutate categories.

URL input is trimmed, limited to 2048 characters, and restricted to absolute HTTP(S) URLs without embedded credentials or whitespace. The backend does not fetch supplied URLs. Availability is handled by the frontend.

## Image processing and storage

The existing CatalogService ImageKit client/configuration is reused. Uploads reuse `compressToWebP` (800px width, Lanczos resize, WebP quality 75), `buildFilename`, and the shared `/zentora` ImageKit upload helper. No new ImageKit credentials/configuration or compression library is introduced.

Category uploads require a successfully decoded JPEG, PNG, GIF, or WebP, at most 8 MB, at most 16 million pixels, dimensions at most 8192, and aspect ratio at most 10:1. These bounds keep decode/resize allocation bounded; GIFs follow the existing first-frame processing behavior. Uploads have a 45-second timeout. Request bodies are capped at 9 MB.

Category uploads fail if ImageKit is unavailable; they do not use the product flow's local fallback. The existing product fallback remains unchanged. If category persistence fails after uploading, the newly uploaded ImageKit file is deleted best-effort with a separate timeout; cleanup failure is logged. Old files are retained when replacing, removing, or deleting a category, matching existing product deletion behavior and avoiding deletion of potentially shared URL assets.

## Admin and storefront

The routed admin category page now supports editing, current-image preview, upload, URL entry, replacement, removal, and leaving the image unchanged. Errors remain visible in the form. Saves invalidate both admin and storefront category queries.

The existing frontend category API types already had optional `image_url`; the category cards now consume it. The shared `CategoryImage` component displays the original fallback while loading, only reveals an image after load, and returns to the fallback on error. Missing/empty/invalid fields never create an image element. Changing the URL remounts the loader and clears previous failure state. Text-only category navigation is unchanged.

## Verification

```powershell
# Backend, with an explicitly disposable LOCAL PostgreSQL cluster:
$env:CATEGORY_TEST_DATABASE_URL = 'postgres://TEST_USER@127.0.0.1:TEST_PORT/TEST_DB?sslmode=disable'
go test ./internal/domain/category ./internal/service/catalog ./internal/handlers/catalog ./internal/repository/postgres ./internal/app

# Frontend:
node --test scripts/category-images.test.mjs
npm run build
```

The PostgreSQL integration test creates/removes a unique schema, applies migration 010 over the original category schema, and exercises the real handlers, services and repository. ImageKit uses the real SDK with a mocked HTTP transport; tests inspect uploaded WebP bytes and folder, successful create/update, URL changes, unchanged images on metadata edits, removal, invalid data, upload failure, rollback cleanup, and response projections. A product regression test verifies existing WebP compression and local fallback. Frontend tests cover URL validation, rendering, load/error transitions, replacement reset, and JSON/multipart admin requests. Live ImageKit account configuration and a visual browser walkthrough remain deployment checks.

## Files

Backend:

- `internal/db/migrations/010_category_images.sql`
- `internal/domain/category/{entity,dto,image,image_test}.go`
- `internal/repository/postgres/category_repo.go`
- `internal/service/catalog/{category,category_image,image_helper,category_image_integration_test,image_helper_test}.go`
- `internal/handlers/catalog/{category,category_image,handler}.go`
- `docs/category-images.md`

Frontend:

- `src/features/admin/catalog/shared/adminCatalogApi.ts`
- `src/features/admin/catalog/categories/components/{CreateCategoryForm,CategoriesTable}.tsx`
- `src/features/admin/catalog/categories/hooks/useCategories.ts`
- `src/features/admin/catalog/categories/pages/AdminCategoriesPage.tsx`
- `src/features/catalog/CategoryImage.tsx`
- `src/features/catalog/categoryImageUrl.ts`
- `src/features/public/home/components/CategoryGrid.tsx`
- `scripts/category-images.test.mjs`
