# Fitness Tracker Backend - Render Deployment

## Deploy to Render

1. **Push to GitHub**:
   ```bash
   git add .
   git commit -m "Add Render deployment"
   git push origin main
   ```

2. **Create Render Service**:
   - Go to [render.com](https://render.com)
   - Click "New +" → "Web Service"
   - Connect your GitHub repository
   - Select `fitness-tracker/backend` as root directory

3. **Configure Service**:
   - **Name**: `fitness-tracker-backend`
   - **Environment**: `Go`
   - **Build Command**: `go build -o main .`
   - **Start Command**: `./main`
   - **Instance Type**: Free

4. **Deploy**: Click "Create Web Service"

## API Endpoints

Once deployed, your API will be available at:
- `https://your-app-name.onrender.com/api/fitness`
- `https://your-app-name.onrender.com/api/fitness/{date}`

## Local Development

```bash
go mod tidy
go run .
```

Server runs on http://localhost:8080