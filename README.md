# Chatbot Demo

A modern React + Vite + TypeScript + Tailwind CSS chatbot demo.

## Project Structure

```
chatbot-demo/
├── src/
│   ├── components/
│   │   └── ChatSidebar.tsx
│   ├── App.tsx
│   ├── main.tsx
│   ├── index.css
│   └── vite-env.d.ts
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.js
└── README.md
```

## Tech Stack

- **React 19** - Latest React with improved performance and features
- **TypeScript** - Type safety and better developer experience
- **Vite** - Lightning-fast development and building
- **Tailwind CSS v4** - Utility-first CSS framework with Vite plugin
- **clsx** - Utility for constructing className strings conditionally

## Features

- ✨ Modern React 19 with functional components and hooks
- 🎨 Tailwind CSS v4 with utility classes (no separate CSS files)
- 📦 Named exports pattern for better refactoring
- 🔧 clsx for clean conditional styling
- ⚡ Vite for fast hot-reload development
- 💪 Full TypeScript support
- 🎯 Clean and minimal configuration

## Prerequisites

- **Node.js** 18+ and yarn (or npm)

## Setup Instructions

```bash
yarn install
```

## Running the Project

```bash
yarn dev
```

The frontend will start on `http://localhost:5173` (Vite default port)

## Development

```bash
yarn dev
```

The frontend will hot-reload on changes.

For production builds:

```bash
yarn build
```

The built files will be in the `dist/` directory.

To preview the production build:

```bash
yarn preview
```

## Project Highlights

### Component Architecture

All components use **named exports** for better IDE support and refactoring:

```tsx
// Export
export function ChatSidebar() { ... }

// Import
import { ChatSidebar } from './components/ChatSidebar'
```

### Styling with Tailwind CSS v4

The project uses Tailwind CSS v4's new Vite plugin approach with utility classes:

```tsx
<div className={clsx(
  'fixed right-0 top-0 w-[400px] h-screen',
  'bg-white shadow-lg',
  isOpen ? 'translate-x-0' : 'translate-x-full'
)}>
  ...
</div>
```

### Configuration

`vite.config.js` uses minimal configuration:

```js
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
})
```

## Next Steps

Ready to integrate with AG-UI protocol or another chatbot backend!

## License

This is a demo project for educational purposes.
