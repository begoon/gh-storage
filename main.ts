import { env } from "node:process";
import { Hono } from "npm:hono";
import { HTTPException } from "npm:hono/http-exception";

import consola from "npm:consola";
import meta from "./deno.json" with { type: "json" };
import { GitHubAPI } from "./github.ts";

const { version } = meta;
console.log(`version ${version}`);

const ME = env.ME;

if (!ME) throw new Error("ME is not defined");

const gh = new GitHubAPI();

const application = new Hono();

application.get("/", (c) => c.text("ha?"));
application.get("/health", (c) => c.json({ version }));

async function timed<T>(f: () => Promise<T>) {
    const start = performance.now();
    const v = await f();
    const elapsed = performance.now() - start;
    return { v, elapsed };
}

application.get("/status", async (c) => {
    const readme = "README.md";

    const exist = await timed(() => gh.exist(readme));
    if (!exist.v) consola.error(`${readme} does not exist`);

    const read = await timed(() => gh.get(readme));
    const raw = await timed(() => gh.raw(readme));

    const ok = read.v?.content === raw.v;
    if (!ok) {
        const n = (v: unknown) => JSON.stringify(v, null, 2);
        consola.error(`read=${n(read.v?.content)} != raw=${n(raw.v)}`);
    }

    const header = typeof raw.v === "string" ? raw.v?.split("\n").at(0) : "?";
    return c.json({
        version,
        exist: exist.elapsed,
        read: read.elapsed,
        raw: raw.elapsed,
        ok,
        header,
    });
});

application.get("/data/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");
    const binary = Boolean(c.req.query("binary"));

    consola.info("get", path, binary);
    const data = await gh.get(path, binary);
    consola.info("get/data", path, data?.size);

    if (!data) return c.notFound();

    const content = data.content;
    return c.body(content);
});

application.get("/raw/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");

    consola.info("raw", path);
    const data = await gh.raw(path);
    const sz = data instanceof ArrayBuffer ? data.byteLength : data?.length;
    consola.info("raw/data", path, sz);

    if (!data) return c.notFound();

    return c.body(data);
});

application.delete("/data/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");

    consola.info("delete", path);
    const deleted = await gh.delete(path);
    if (!deleted) throw new HTTPException(500, { message: "failed to delete" });

    return c.body("deleted");
});

application.post("/data/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");
    const data = await c.req.text();

    consola.info("create", path);
    const created = await gh.create(path, data);
    if (!created) throw new HTTPException(500, { message: "failed to create" });

    return c.body("created");
});

application.put("/data/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");
    const data = await c.req.text();

    consola.info("update", path);
    const updated = await gh.commit(path, data);
    if (!updated) throw new HTTPException(500, { message: "failed to update" });

    return c.body("updated");
});

application.on("HEAD", "/data/:path{.+$}", async (c) => {
    const headers = c.req.header("ME");
    if (headers !== ME) throw new HTTPException(403, { message: "forbidden" });

    const path = c.req.param("path");

    consola.info("head", path);
    const exist = await gh.exist(path);
    if (!exist) return c.notFound();
    return c.body("exist");
});

const port = Number(env.PORT) || 8000;
Deno.serve(
    { port, onListen: () => console.log(`listening on ${port}`) },
    application.fetch,
);
