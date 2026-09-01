import type { JobTypeEnum } from "@/lib/types";

export function JobStatusBadges({ jobs }: { jobs: JobTypeEnum[] }) {
  if (jobs.length === 0) {
    return null;
  }

  return (
    <span
      className="pointer-events-none absolute left-2 bottom-2 inline-flex items-center rounded bg-black/75 px-1 py-0.5 text-zinc-100"
      title="Processing"
      aria-label="Processing"
    >
      <svg viewBox="0 0 24 24" className="h-3 w-3 fill-current" aria-hidden>
        <path d="M12 2a1 1 0 0 1 1 1v1.07a7.002 7.002 0 0 1 4.93 4.93H19a1 1 0 1 1 0 2h-1.07A7.002 7.002 0 0 1 13 15.93V17a1 1 0 1 1-2 0v-1.07A7.002 7.002 0 0 1 6.07 11H5a1 1 0 1 1 0-2h1.07A7.002 7.002 0 0 1 11 4.07V3a1 1 0 0 1 1-1Zm0 5a5 5 0 1 0 0 10 5 5 0 0 0 0-10Z" />
      </svg>
    </span>
  );
}
