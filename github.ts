import { Buffer } from "node:buffer";
import { env } from "node:process";
import consola from "npm:consola";

consola.options.formatOptions.columns = 0;
consola.options.formatOptions.compact = false;

export type Data = {
    content: string | Uint8Array;
    sha: string;
    size: number;
    type: string;
    encoding: string;
    name: string;
    download_url: string;
    url: string;
    git_url: string;
    html_url: string;
};

const GH = `https://api.github.com/repos`;
const GH_STORAGE = "https://raw.githubusercontent.com";

const now = () => new Date().toISOString().replace("T", " ").replace("Z", "");

export class GitHubAPI {
    #token: string | undefined;
    #contents: string;
    #raw: string;

    constructor() {
        const { GITHUB_TOKEN, GITHUB_ACCOUNT, GITHUB_REPO } = env;
        this.#token = GITHUB_TOKEN;
        this.#contents = `${GH}/${GITHUB_ACCOUNT}/${GITHUB_REPO}/contents`;
        this.#raw = `${GH_STORAGE}/${GITHUB_ACCOUNT}/${GITHUB_REPO}/main`;
    }

    headers = () => ({
        Authorization: `token ${this.#token}`,
        Accept: "application/vnd.github.v3+json",
    });

    error(message: string, cause: string, response?: Response): undefined {
        if (response) {
            const { ok, status, statusText } = response;
            consola.error(message, {
                cause,
                ok,
                status,
                statusText,
            });
        } else {
            consola.error(message, { cause });
        }
        return undefined;
    }

    async exist(path: string): Promise<boolean> {
        try {
            const url = `${this.#contents}/${path}`;
            const response = await fetch(url, { headers: this.headers() });
            if (response.ok) return true;
            this.error("exist", url, response);
        } catch (e) {
            this.error("exist", e.toString());
        }
        return false;
    }

    async get(path: string, binary?: boolean): Promise<Data | undefined> {
        try {
            const url = `${this.#contents}/${path}`;
            const response = await fetch(url, { headers: this.headers() });
            if (!response.ok) {
                return this.error("get: not found", url, response);
            }
            const data = (await response.json()) as Data;
            const bytes = Buffer.from(data.content as string, "base64");
            data.content = binary ? bytes : new TextDecoder().decode(bytes);
            return data;
        } catch (e) {
            return this.error("get", e.toString());
        }
    }

    async raw(
        path: string,
        binary?: boolean,
    ): Promise<ArrayBuffer | string | undefined> {
        try {
            const url = `${this.#raw}/${path}`;
            const response = await fetch(url, { headers: this.headers() });
            if (!response.ok) return this.error("raw", url, response);
            const data = binary
                ? await response.arrayBuffer()
                : await response.text();
            return data;
        } catch (e) {
            return this.error("raw", e.toString());
        }
    }

    async delete(path: string, sha?: string): Promise<boolean> {
        try {
            if (!sha) {
                const data = await this.get(path);
                if (!data) return false;
                sha = data.sha;
            }
            const message = `delete ${now()}`;
            const url = `${this.#contents}/${path}`;
            console.log("delete", url);
            const response = await fetch(url, {
                method: "DELETE",
                headers: this.headers(),
                body: JSON.stringify({ message, sha }),
            });
            if (response.ok) return true;
            this.error("delete", url, response);
        } catch (e) {
            this.error("delete", e.toString());
        }
        return false;
    }

    async commit(
        path: string,
        data: string | Uint8Array,
        sha?: string,
    ): Promise<boolean> {
        try {
            const message = `commit ${now()}, ${path}`;
            const raw = data instanceof Uint8Array
                ? data
                : new TextEncoder().encode(data);
            const content = Buffer.from(raw).toString("base64");

            const url = `${this.#contents}/${path}`;

            if (!sha) {
                const data = await this.get(path);
                if (!data) return false;
                sha = data.sha;
            }
            const result = await fetch(url, {
                method: "PUT",
                headers: this.headers(),
                body: JSON.stringify({ message, content, sha }),
            });
            if (result.ok) return true;
            this.error("commit", url, result);
        } catch (e) {
            this.error("commit", e.toString());
        }
        return false;
    }

    async create(path: string, data: string): Promise<boolean> {
        try {
            const message = `create ${now()}`;
            const content = Buffer.from(
                new TextEncoder().encode(data),
            ).toString("base64");
            const url = `${this.#contents}/${path}`;
            const result = await fetch(url, {
                method: "PUT",
                headers: this.headers(),
                body: JSON.stringify({ message, content }),
            });
            if (result.ok) return true;
            this.error("create", url, result);
        } catch (e) {
            this.error("create", e.toString());
        }
        return false;
    }
}
