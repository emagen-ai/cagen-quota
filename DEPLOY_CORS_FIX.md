# CORS Fix Deployment Guide

## Changes Made

1. **Created proper CORS middleware** (`internal/middleware/cors.go`):
   - Configurable allowed origins
   - Proper preflight request handling
   - Support for credentials
   - Configurable headers and methods

2. **Updated configuration** (`internal/config/config.go`):
   - Added `AllowedOrigins` field
   - Environment variable: `ALLOWED_ORIGINS` (comma-separated list)
   - Default: `https://cyberagent-frontend.vercel.app,http://localhost:3000,http://localhost:3001`

3. **Updated main.go**:
   - Imported the new middleware package
   - Replaced inline CORS headers with proper middleware
   - Parse allowed origins from environment variable

## Deployment Steps

### 1. Update Railway Environment Variables

Add or update the following environment variable in your Railway service:

```bash
ALLOWED_ORIGINS=https://cyberagent-frontend.vercel.app,http://localhost:3000,http://localhost:3001
```

You can add more origins as needed, separated by commas.

### 2. Deploy to Railway

Option A - Using Railway CLI:
```bash
# Make sure you're in the project directory
cd /home/kiwi/workspace/cagen-quato/cagen-quota

# Link to your Railway project (if not already linked)
railway link

# Deploy
railway up
```

Option B - Using Git:
```bash
# Commit the changes
git add .
git commit -m "Fix CORS configuration for frontend access"

# Push to trigger Railway deployment
git push
```

### 3. Verify Deployment

After deployment, test the CORS configuration:

1. Check the service logs in Railway dashboard for:
   ```
   CORS allowed origins: [https://cyberagent-frontend.vercel.app http://localhost:3000 http://localhost:3001]
   ```

2. Test from the frontend:
   - The API calls should no longer get CORS errors
   - Check browser DevTools Network tab for proper CORS headers

## Additional Notes

- The CORS middleware allows credentials by default (`Access-Control-Allow-Credentials: true`)
- Preflight requests are handled with a 24-hour cache (`Access-Control-Max-Age: 86400`)
- All standard HTTP methods are allowed (GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD)
- Common headers are allowed including Authorization and X-Request-ID

## Troubleshooting

If CORS issues persist:

1. Check Railway logs for the actual allowed origins being used
2. Verify the frontend is sending requests from an allowed origin
3. Check if the frontend needs any specific headers that aren't in the allowed list
4. Update `ALLOWED_ORIGINS` environment variable and redeploy if needed