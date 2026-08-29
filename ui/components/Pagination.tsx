import { PAGE_SIZE } from "@/lib/config";

export function Pagination({
  page,
  total,
  onPage,
}: {
  page: number;
  total: number;
  onPage: (page: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  if (totalPages <= 1) {
    return null;
  }

  const uiPage = page + 1;
  const numbers = pageNumbers(uiPage, totalPages);

  return (
    <nav
      aria-label="Pagination"
      className="flex flex-wrap items-center justify-center gap-1 py-6"
    >
      <PageBtn
        disabled={page <= 0}
        onClick={() => onPage(page - 1)}
        label="Previous"
      >
        Prev
      </PageBtn>
      {numbers.map((n, i) =>
        n === "…" ? (
          <span key={`e${i}`} className="px-2 text-sm text-muted">
            …
          </span>
        ) : (
          <PageBtn
            key={n}
            current={n === uiPage}
            onClick={() => onPage(n - 1)}
            label={`Page ${n}`}
          >
            {n}
          </PageBtn>
        ),
      )}
      <PageBtn
        disabled={page >= totalPages - 1}
        onClick={() => onPage(page + 1)}
        label="Next"
      >
        Next
      </PageBtn>
    </nav>
  );
}

function PageBtn({
  children,
  onClick,
  disabled,
  current,
  label,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  current?: boolean;
  label: string;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-current={current ? "page" : undefined}
      disabled={disabled}
      onClick={onClick}
      className={`min-w-9 rounded-md px-2.5 py-1.5 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:opacity-40 ${
        current
          ? "bg-accent font-medium text-accent-fg"
          : "text-muted hover:bg-surface-hover hover:text-foreground"
      }`}
    >
      {children}
    </button>
  );
}

function pageNumbers(current: number, total: number): (number | "…")[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const set = new Set([1, total, current - 1, current, current + 1]);
  const nums = [...set]
    .filter((n) => n >= 1 && n <= total)
    .sort((a, b) => a - b);
  const out: (number | "…")[] = [];
  for (let i = 0; i < nums.length; i++) {
    if (i > 0 && nums[i] - nums[i - 1] > 1) {
      out.push("…");
    }
    out.push(nums[i]);
  }
  return out;
}
