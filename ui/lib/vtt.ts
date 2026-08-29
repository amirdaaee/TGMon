export type SpriteCue = {
  start: number;
  end: number;
  x: number;
  y: number;
  w: number;
  h: number;
};

function parseVttTime(value: string): number {
  const parts = value.trim().split(":");
  if (parts.length === 3) {
    return Number(parts[0]) * 3600 + Number(parts[1]) * 60 + Number(parts[2]);
  }
  if (parts.length === 2) {
    return Number(parts[0]) * 60 + Number(parts[1]);
  }
  return Number(value);
}

function parseXywh(payload: string): { x: number; y: number; w: number; h: number } | null {
  const match = payload.match(/#xywh=(-?\d+),(-?\d+),(-?\d+),(-?\d+)/i);
  if (!match) {
    return null;
  }
  return {
    x: Number(match[1]),
    y: Number(match[2]),
    w: Number(match[3]),
    h: Number(match[4]),
  };
}

export function parseSpriteVtt(text: string): SpriteCue[] {
  const cues: SpriteCue[] = [];
  const blocks = text.replace(/^\uFEFF/, "").split(/\n\s*\n/);

  for (const block of blocks) {
    const lines = block
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((line) => line && !line.startsWith("NOTE") && line !== "WEBVTT");
    if (lines.length === 0) {
      continue;
    }

    const timingLine = lines.find((line) => line.includes("-->"));
    if (!timingLine) {
      continue;
    }
    const [startRaw, endRaw] = timingLine.split("-->");
    const start = parseVttTime(startRaw.replace(/[^\d:.]/g, ""));
    const end = parseVttTime((endRaw ?? "").split(/\s/)[0].replace(/[^\d:.]/g, ""));
    if (!Number.isFinite(start) || !Number.isFinite(end)) {
      continue;
    }

    const payload = lines.find((line) => line !== timingLine && parseXywh(line));
    const xywh = payload ? parseXywh(payload) : null;
    if (!xywh) {
      continue;
    }
    cues.push({ start, end, ...xywh });
  }

  return cues.sort((a, b) => a.start - b.start);
}

export function findSpriteCue(
  cues: SpriteCue[],
  time: number,
): SpriteCue | null {
  if (cues.length === 0) {
    return null;
  }
  for (const cue of cues) {
    if (time >= cue.start && time < cue.end) {
      return cue;
    }
  }
  if (time < cues[0].start) {
    return cues[0];
  }
  return cues[cues.length - 1];
}
