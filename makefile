run:
	deno run --config tsconfig.json --allow-read --allow-env --allow-net index.ts
test:
	deno test
format:
	deno fmt
debug:
	denon run -A --config tsconfig.json --allow-read --allow-env --allow-net --inspect-brk index.ts
cache:
	deno cache index.ts