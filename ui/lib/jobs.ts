import type { JobReqDoc, JobTypeEnum } from "./types";

export function jobsByMediaId(jobs: JobReqDoc[]): Map<string, JobTypeEnum[]> {
  const map = new Map<string, JobTypeEnum[]>();
  for (const job of jobs) {
    const mediaId = String(job.MediaID);
    const existing = map.get(mediaId);
    if (existing) {
      if (!existing.includes(job.Type)) {
        existing.push(job.Type);
      }
    } else {
      map.set(mediaId, [job.Type]);
    }
  }
  return map;
}
