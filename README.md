# Chatbot Demo with AG-UI Client

A modern React + Vite chatbot application that connects to agents supporting the AG-UI protocol.

## Features

- 🤖 **AG-UI Protocol Integration** - Connects to any AG-UI compatible agent
- 🔌 **Observable Streaming** - Real-time responses via Server-Sent Events
- 💚 **Connection Health Check** - Automatic status monitoring
- 🎨 **Modern UI** - React 19 + Tailwind CSS v4 + shadcn/ui components
- 🧩 **AI SDK Elements** - Enhanced conversation UI with auto-scroll
- ⚡ **Fast Development** - Vite with hot reload
- 💪 **TypeScript** - Full type safety

## Prerequisites

- Node.js 18+
- An AG-UI compatible agent running (default: `localhost:8000`)

## Quick Start

```bash
# Install dependencies
yarn install

# Start development server
yarn dev
```

Visit `http://localhost:5173`

## Configuration

The application proxies requests to avoid CORS issues. Update `vite.config.js` if your agent runs on a different port:

```js
server: {
  proxy: {
    '/api/agent': {
      target: 'http://localhost:8000',  // Change this if needed
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/api\/agent/, ''),
    },
  },
}
```

## Adding UI Components

This project uses [shadcn/ui](https://ui.shadcn.com/) for components and [AI SDK Elements](https://sdk.vercel.ai/docs/ai-sdk-ui/ai-sdk-elements) for conversation UI.

### Adding shadcn components:
```bash
npx shadcn@latest add <component-name>
```

### Adding AI Elements components:
```bash
npx shadcn@latest add @ai-elements/<component-name>
```

Example:
```bash
npx shadcn@latest add @ai-elements/conversation
```

## Build for Production

```bash
# Build
yarn build

# Preview
yarn preview
```

## Tech Stack

- React 19
- TypeScript
- Vite
- Tailwind CSS v4
- shadcn/ui - Component library
- AI SDK Elements - Conversation components
- @ag-ui/core (types for AG-UI protocol)

## License

MIT
