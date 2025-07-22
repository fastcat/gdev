import { createServer } from "node:http";
import {} from 'node:process'

function mp<T>() {
	let resolve!: (value: T | PromiseLike<T>) => void;
	let reject!: (reason?: any) => void;
	const p = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return Object.assign(p, {
		resolve: resolve,
		reject: reject,
	});
}

async function main() {
	const server = createServer();
	let sp = mp<void>();
	server.on('request', (req, res) => {
		if (req.url === '/') {
			if (req.method === 'GET') {
				res.writeHead(200, { "content-type": "text/plain" });
				res.end("Hello World!\nThis is svc2!\n");
			} else {
				res.writeHead(405);
				res.end();
			}
		} else {
			res.writeHead(404);
			res.end();
		}
	});
	server.on('error', (err) => sp.reject(err));

	let port = 8081;
	if (process.env.PORT) {
		port = parseInt(process.env.PORT, 10);
	}

	process.on('SIGINT', () => sp.resolve())
	process.on('SIGTERM', () => sp.resolve());

	console.log(`Starting server on port ${port}...`);
	server.listen(port);
	await sp;
	sp = mp<void>();
	server.close((err) => err ? sp.reject(err) : sp.resolve());
	await sp;
}

try {
	await main();
} catch (err) {
	console.error(err);
	process.exit(1);
}
