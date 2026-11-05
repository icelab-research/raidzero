import { basename, join } from "node:path";
import { mkdirSync } from "node:fs";
import { randomUUID } from "node:crypto";
import { sfVoice } from "../client.ts";
import { config } from "../config.ts";
import { SfVoiceMediaError } from "../../sdk/src/errors.js";

const TMP_DIR = join(import.meta.dir, "..", "tmp");
mkdirSync(TMP_DIR, { recursive: true });

function apiError(e: unknown): Response {
  if (e instanceof SfVoiceMediaError) {
    return Response.json({ error: e.message, code: e.code }, { status: e.status });
  }
  const msg = e instanceof Error ? e.message : "unknown error";
  return Response.json({ error: msg }, { status: 500 });
}

export const ingestRoutes = {
  // POST /api/ingest/youtube  { url: string }
  // downloads the video with yt-dlp, serves it from /media/:file,
  // then submits the local URL to the ingest API.
  // requires yt-dlp in PATH: https://github.com/yt-dlp/yt-dlp
  youtube: async (req: Request): Promise<Response> => {
    const body = await req.json() as { url?: string };
    const url = body.url?.trim();
    if (!url) return Response.json({ error: "url is required" }, { status: 400 });
    if (!url.includes("youtube.com") && !url.includes("youtu.be")) {
      return Response.json({ error: "not a youtube url" }, { status: 400 });
    }

    const id = randomUUID();
    const outPath = join(TMP_DIR, `${id}.mp4`);

    // prefer 720p mp4 to keep downloads reasonably sized
    const dl = Bun.spawn(
      ["yt-dlp", "-f", "best[height<=720][ext=mp4]/best[ext=mp4]/best", "-o", outPath, url],
      { stdout: "ignore", stderr: "pipe" }
    );
    await dl.exited;
    if (dl.exitCode !== 0) {
      const stderr = await new Response(dl.stderr).text();
      const hint = stderr.includes("not found") || stderr.includes("command not found")
        ? "yt-dlp is not installed — run: pip install yt-dlp"
        : stderr.trim();
      return Response.json({ error: "yt-dlp failed", detail: hint }, { status: 422 });
    }

    // grab the video title for metadata
    const titleProc = Bun.spawn(["yt-dlp", "--get-title", "--no-playlist", url], { stdout: "pipe", stderr: "ignore" });
    await titleProc.exited;
    const title = (await new Response(titleProc.stdout).text()).trim() || undefined;

    try {
      const mediaUrl = `${config.selfUrl}/media/${id}.mp4`;
      const result = await sfVoice.ingest({ source: "url", url: mediaUrl, media_type: "video", metadata: { title } });
      return Response.json(result);
    } catch (e) {
      return apiError(e);
    }
  },

  // POST /api/ingest/file  (multipart: file, title?)
  // saves the uploaded file to tmp/, serves it from /media/:file,
  // then submits to the ingest API via `source: "url"`.
  //
  // note: the API supports direct multipart uploads on `POST /v1/ingest` (and
  // browser-direct presigned uploads on `/v1/ingest/sessions/*`); this demo
  // keeps the self-host detour because it's structured around a local file
  // server. swapping to direct multipart is a small follow-up.
  file: async (req: Request): Promise<Response> => {
    const form = await req.formData().catch(() => null);
    if (!form) return Response.json({ error: "expected multipart/form-data" }, { status: 400 });

    const file = form.get("file") as File | null;
    if (!file) return Response.json({ error: "missing file field" }, { status: 400 });

    const ext = file.name.split(".").pop()?.toLowerCase() ?? "mp4";
    const id = randomUUID();
    const filename = `${id}.${ext}`;
    const outPath = join(TMP_DIR, filename);

    await Bun.write(outPath, file);

    const title = (form.get("title") as string | null) ?? file.name;
    const mediaType = file.type.startsWith("audio/") ? "audio" as const : "video" as const;

    try {
      const mediaUrl = `${config.selfUrl}/media/${filename}`;
      const result = await sfVoice.ingest({ source: "url", url: mediaUrl, media_type: mediaType, metadata: { title } });
      return Response.json(result);
    } catch (e) {
      return apiError(e);
    }
  },

  // GET /api/tasks/:id
  task: async (req: Request): Promise<Response> => {
    const id = basename(new URL(req.url).pathname);
    try {
      const task = await sfVoice.getTask(id);
      return Response.json(task);
    } catch (e) {
      return apiError(e);
    }
  },

  // GET /media/:filename — serves temp files to the sf-voice backend
  media: async (req: Request): Promise<Response> => {
    // basename strips any path separators to prevent traversal
    const filename = basename(new URL(req.url).pathname.slice("/media/".length));
    if (!filename) return new Response("not found", { status: 404 });

    const file = Bun.file(join(TMP_DIR, filename));
    if (!(await file.exists())) return new Response("not found", { status: 404 });
    return new Response(file);
  },
};
