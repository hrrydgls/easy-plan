export const pad = (n: number) => String(n).padStart(2, '0');

/** "YYYY-MM-DD" for a local year, zero-based month, and day. */
export const isoDay = (y: number, m: number, d: number) => `${y}-${pad(m + 1)}-${pad(d)}`;

export const fmt = (d: Date, o: Intl.DateTimeFormatOptions) => d.toLocaleDateString(undefined, o);

/** ISO-8601 week number (weeks start Monday; week 1 holds the first Thursday). */
export function isoWeek(d: Date): number {
	const t = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
	const day = t.getUTCDay() || 7;
	t.setUTCDate(t.getUTCDate() + 4 - day);
	const y0 = new Date(Date.UTC(t.getUTCFullYear(), 0, 1));
	return Math.ceil(((t.getTime() - y0.getTime()) / 86400000 + 1) / 7);
}

/** Short weekday names starting Monday, in the viewer's locale. */
export const weekdayNames = () =>
	Array.from({ length: 7 }, (_, i) => fmt(new Date(2024, 0, 1 + i), { weekday: 'short' }));
