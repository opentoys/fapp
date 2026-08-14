// The backend stores timestamps in the server's default timezone and sends
// them as RFC3339 with that offset. Render the wall-clock portion as-is so the
// display matches what was stored — never use toISOString() here, which shifts
// to UTC and misrepresents the server-local time.
export function fmtDate(s: string): string {
  const m = s.match(/^(\d{4}-\d{2}-\d{2})T?(\d{2}:\d{2}:\d{2})/)
  return m ? `${m[1]} ${m[2]}` : s
}
