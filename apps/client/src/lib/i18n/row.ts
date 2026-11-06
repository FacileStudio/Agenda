import type { Message } from '@facile/muse';

/** One message in the four locales, in the order en, fr, es, de. */
export type Row = readonly [Message, Message, Message, Message];
