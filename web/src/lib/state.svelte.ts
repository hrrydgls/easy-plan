import { api, ApiError, type User } from './api';

// ---------- Session ----------

export const session = $state<{ user: User | null; checked: boolean }>({ user: null, checked: false });

/** Loads the current user once. Resolves to null when logged out. */
export async function loadSession(): Promise<User | null> {
	if (session.checked) return session.user;
	try {
		session.user = await api.me();
	} catch (e) {
		if (!(e instanceof ApiError) || e.status !== 401) throw e;
		session.user = null;
	}
	session.checked = true;
	return session.user;
}

// ---------- Theme ----------

export type Theme = 'light' | 'dark';
export const theme = $state<{ value: Theme }>({
	value: typeof document !== 'undefined' && document.documentElement.classList.contains('dark') ? 'dark' : 'light'
});

export function setTheme(t: Theme) {
	theme.value = t;
	document.documentElement.classList.toggle('dark', t === 'dark');
	try {
		localStorage.setItem('ep-theme', t);
	} catch {
		/* storage unavailable */
	}
}

// ---------- Toast ----------

export const toast = $state<{ text: string; id: number }>({ text: '', id: 0 });
let toastTimer: ReturnType<typeof setTimeout>;
export function showToast(text: string) {
	toast.text = text;
	toast.id++;
	clearTimeout(toastTimer);
	toastTimer = setTimeout(() => (toast.text = ''), 2800);
}

// ---------- Last opened calendar ----------

export function rememberCalendar(id: number) {
	try {
		localStorage.setItem('ep-last-calendar', String(id));
	} catch {
		/* ignore */
	}
}
export function lastCalendar(): number | null {
	try {
		const v = Number(localStorage.getItem('ep-last-calendar'));
		return Number.isFinite(v) && v > 0 ? v : null;
	} catch {
		return null;
	}
}
