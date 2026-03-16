# Deploying to GitHub Pages

Since GitHub Pages only supports static websites, you'll need a slightly different approach. Here are two options:

## Option 1: Frontend Only (Demo Mode)

Deploy just the React webapp to GitHub Pages with a mock/demo backend.

### Setup Steps:

1. **Push to GitHub** (if not already done):
```bash
# From your project root
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/spellchecker.git
git push -u origin main
```

2. **Create GitHub Actions workflow** (`.github/workflows/deploy.yml`):

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          
      - name: Install dependencies
        run: |
          cd webapp
          npm ci
          
      - name: Build
        run: |
          cd webapp
          npm run build
        env:
          VITE_API_URL: https://your-backend-url.com  # If you have a hosted backend
          
      - name: Setup Pages
        uses: actions/configure-pages@v4
        
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './webapp/dist'
          
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

3. **Enable GitHub Pages**:
   - Go to your repository on GitHub
   - Click Settings → Pages
   - Under "Build and deployment", select "GitHub Actions"
   - Save

4. **Update vite.config.ts** for GitHub Pages:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/spellchecker/',  // Important: match your repo name
  server: {
    port: 3000,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
```

## Option 2: Full Stack (Recommended)

Host the backend separately and connect the frontend to it.

### Backend Hosting Options:

**A. Render.com (Free)**
```bash
# Create render.yaml in project root
services:
  - type: web
    name: spellchecker-api
    env: go
    buildCommand: go build -o server ./cmd/server
    startCommand: ./server
    envVars:
      - key: LISTEN_ADDR
        value: :10000
      - key: DICT_PATH
        value: /dictionaries/en_gb.txt
```

**B. Railway.app (Free tier)**
- Connect your GitHub repo
- It auto-detects the Go project
- Deploys automatically

**C. Fly.io (Free tier)**
```bash
# Install flyctl
# Create fly.toml
cat > fly.toml << 'EOF'
app = "spellchecker-api"
primary_region = "lhr"

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
EOF

fly deploy
```

### Connecting Frontend to Backend:

Update your webapp to use the hosted backend:

```typescript
// webapp/.env.production
VITE_API_URL=https://spellchecker-api.fly.dev
```

Or hardcode it in the API hook:

```typescript
// webapp/src/hooks/useSpellCheck.ts
const API_BASE = import.meta.env.VITE_API_URL || 'https://spellchecker-api.fly.dev';
```

## Important Notes:

1. **GitHub Pages Limitations**:
   - Only serves static files
   - No backend/Go server support
   - Must host API separately

2. **CORS Issues**:
   If your frontend and backend are on different domains, ensure your Go backend allows CORS:
   ```go
   // Already implemented in handler.go!
   w.Header().Set("Access-Control-Allow-Origin", "*")
   ```

3. **Environment Variables**:
   - Use `.env.production` for production values
   - Use `.env.development` for local development

## Quick Deploy Checklist:

- [ ] Push code to GitHub
- [ ] Add GitHub Actions workflow
- [ ] Enable GitHub Pages (GitHub Actions source)
- [ ] Update `vite.config.ts` with correct `base` path
- [ ] Set environment variables in GitHub Settings → Secrets
- [ ] Deploy backend separately (if using full stack)
- [ ] Update frontend API URL to point to hosted backend
- [ ] Test the deployed site

## Demo Mode (No Backend):

If you just want to show the UI without a working backend:

```typescript
// In useSpellCheck.ts, add a mock mode:
export function useSpellCheck() {
  const [isLoading, setIsLoading] = useState(false);
  
  const checkText = useCallback(async (request) => {
    setIsLoading(true);
    
    // Mock response for demo
    await new Promise(r => setTimeout(r, 500));
    
    return {
      misspellings: [
        {
          word: "teh",
          offset: 4,
          line: 1,
          column: 5,
          suggestions: [{ word: "the", edit_distance: 1 }]
        }
      ],
      repeated_words: [],
      capitalisation_issues: [],
      token_count: 5,
      checked_count: 5,
      elapsed_ms: 0.5
    };
  }, []);
  
  return { checkText, isLoading, error: null };
}
```

This lets you deploy just the frontend to GitHub Pages for UI demos!

## Troubleshooting:

**Blank page after deploy?**
- Check browser console for 404 errors
- Verify `base` path in vite.config.ts matches your repo name
- Ensure all assets are in the `dist` folder

**CORS errors?**
- Backend must send CORS headers
- Check that API URL is correct in environment variables

**Build fails?**
- Check Actions logs in GitHub
- Ensure `npm ci` works locally
- Verify Node version compatibility
