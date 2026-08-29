export function BootScreen() {
  return (
    <div className="flex min-h-full flex-1 items-center justify-center bg-background">
      <div
        className="h-8 w-8 animate-spin rounded-full border-2 border-zinc-700 border-t-accent"
        aria-label="Loading"
      />
    </div>
  );
}
