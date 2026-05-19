// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

// Tipagem do Web Component emoji-picker-element para o TypeScript não reclamar
declare namespace svelteHTML {
	interface IntrinsicElements {
		'emoji-picker': {
			class?: string;
			'on:emoji-click'?: (e: CustomEvent) => void;
		};
	}
}

export {};
