// Thin JSON client for the Easy Plan Go API.

export type CountMode = 'none' | 'month' | 'year';

export interface User {
	id: number;
	name: string;
	email: string;
	createdAt: number;
}

export interface Calendar {
	id: number;
	name: string;
	createdAt: number;
}

export interface EventType {
	id: number; // 0 for a not-yet-saved type
	name: string;
	emoji: string;
	count: CountMode;
	position?: number;
}

export interface CalendarDetail extends Calendar {
	eventTypes: EventType[];
	days: Record<string, number[]>;
}

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
	}
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	let res: Response;
	try {
		res = await fetch(`/api${path}`, {
			method,
			credentials: 'same-origin',
			headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	} catch {
		throw new ApiError(0, "Can't reach the server. Check your connection and try again.");
	}
	if (res.status === 204) return undefined as T;
	let data: any = null;
	try {
		data = await res.json();
	} catch {
		/* non-JSON body */
	}
	if (!res.ok) throw new ApiError(res.status, data?.error ?? `The server answered ${res.status}.`);
	return data as T;
}

export const api = {
	config: () => request<{ registration: boolean }>('GET', '/config'),
	me: () => request<User>('GET', '/me'),
	register: (name: string, email: string, password: string) =>
		request<User>('POST', '/auth/register', { name, email, password }),
	login: (email: string, password: string) => request<User>('POST', '/auth/login', { email, password }),
	logout: () => request<void>('POST', '/auth/logout', {}),

	calendars: () => request<Calendar[]>('GET', '/calendars'),
	createCalendar: (name: string, starter: boolean) =>
		request<Calendar>('POST', '/calendars', { name, starter }),
	calendar: (id: number) => request<CalendarDetail>('GET', `/calendars/${id}`),
	renameCalendar: (id: number, name: string) => request<Calendar>('PATCH', `/calendars/${id}`, { name }),
	deleteCalendar: (id: number) => request<void>('DELETE', `/calendars/${id}`),
	saveEventTypes: (id: number, types: EventType[]) =>
		request<CalendarDetail>('PUT', `/calendars/${id}/event-types`, types),
	setDay: (id: number, day: string, eventTypeIds: number[]) =>
		request<{ day: string; eventTypeIds: number[] }>('PUT', `/calendars/${id}/days/${day}`, { eventTypeIds })
};
