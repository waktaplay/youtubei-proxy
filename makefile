run:
	deno run --config tsconfig.json --allow-read --allow-write --allow-net  index.ts
test:
	deno test
format:
	deno fmt
debug:
	denon run -A --allow-read --allow-write --allow-net --inspect-brk index.ts
bundle:
	rm -rf build/
	mkdir build
	deno bundle index.ts build/index	