# Spell Checker Webapp

A React-based web application for spell checking text with profile management.

## Features

- **Text Editor**: Large content-editable area for pasting and editing text
- **Spell Checking**: Click "Check Spelling" to analyze text
- **Inline Highlights**: Visual indicators for misspellings (red), repeated words (yellow), and capitalisation issues (blue)
- **Issues Sidebar**: Right panel showing all issues with quick-fix buttons
- **Profile Management**: Create and manage custom profiles with allowed terms
- **Undo Support**: Undo suggestion applications
- **Settings**: Toggle auto-check mode and configure debounce delay

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **State Management**: Zustand
- **Icons**: Lucide React

## Development

### Prerequisites

- Node.js 18+
- npm or yarn
- Go backend running on port 8080

### Setup

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The webapp will be available at http://localhost:3000

### Build

```bash
npm run build
```

Output will be in the `dist/` directory.

## Environment Variables

- `VITE_API_URL`: URL of the spell checker backend API (default: http://localhost:8080)

## Docker

The webapp is included in the main docker-compose.yaml:

```bash
docker-compose up
```

This will start both the backend and frontend services.

## Project Structure

```
webapp/
├── src/
│   ├── components/
│   │   ├── Editor/           # Text editor component
│   │   ├── IssuesPanel/      # Issues sidebar
│   │   ├── ProfileManager/   # Profile management UI
│   │   ├── Layout/           # Header, settings panel
│   │   └── ui/               # Reusable UI components
│   ├── hooks/                # API hooks
│   ├── stores/               # Zustand stores
│   ├── types/                # TypeScript types
│   └── utils/                # Utility functions
├── public/
└── dist/                     # Build output
```

## API Integration

The webapp communicates with the Go backend via these endpoints:

- `POST /check` - Check text for misspellings
- `GET /profiles` - List all profiles
- `POST /profiles` - Create new profile
- `PUT /profiles/:id` - Update profile
- `DELETE /profiles/:id` - Delete profile
- `POST /profiles/:id/terms` - Add term to profile
- `DELETE /profiles/:id/terms/:term` - Remove term from profile
