# LiveReload + Tailwind + Golang

## Installation

```bash
cd web
npm install
```

## Developing Pages

Best to use two consoles

```bash
npm run dev
```

```bash
go build ./cmd/dev-server/
./dev-server
2021/07/23 19:22:53 dev-server started on port  :8080
```

The page should open automatically to the /index template

## Building for production

```bash
npm run build

> build
> NODE_ENV=production ./node_modules/tailwindcss/lib/cli.js -i ./static/tailwind.css -o ./static/main.css --jit --minify


warn - You have enabled the JIT engine which is currently in preview.
warn - Preview features are not covered by semver, may introduce breaking changes, and can change at any time.

Done in 189ms.
```
