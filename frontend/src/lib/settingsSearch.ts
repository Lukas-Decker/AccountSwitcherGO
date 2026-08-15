/*
  Search plumbing for the settings grid.

  A row knows its own text, so matching happens where the text lives instead of
  in a central table that would drift out of date. Rows report whether they
  matched to the card above them, and cards report the same to the page, so an
  empty card and an empty page can hide themselves without anyone keeping a
  list of what is where.
*/

import { getContext, hasContext, setContext } from "svelte";
import { derived, readable, writable, type Readable } from "svelte/store";

const QUERY_KEY = "settings:search-query";
const CARD_TITLE_KEY = "settings:card-title";
const ROW_REGISTRY_KEY = "settings:row-registry";
const CARD_REGISTRY_KEY = "settings:card-registry";

const EMPTY_QUERY: Readable<string> = readable("");

/** Lowercase, strip accents and collapse whitespace so "Ubersicht" finds "Übersicht". */
export function normalizeSearchText(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLocaleLowerCase()
    .replace(/\s+/g, " ")
    .trim();
}

/** True when every whitespace-separated term of the query appears in the text. */
export function matchesSettingsQuery(text: string, query: string): boolean {
  const needle = normalizeSearchText(query);
  if (!needle) {
    return true;
  }
  const haystack = normalizeSearchText(text);
  return needle.split(" ").every((term) => haystack.includes(term));
}

export interface MatchRegistry {
  /** True when at least one registered member matched the current query. */
  anyMatched: Readable<boolean>;
  /** True while nothing has registered, so a container can fall back to its own text. */
  empty: Readable<boolean>;
  report: (id: string, matched: boolean) => void;
  remove: (id: string) => void;
}

export function createMatchRegistry(): MatchRegistry {
  const members = writable<Map<string, boolean>>(new Map());

  return {
    anyMatched: derived(members, ($members) => {
      for (const matched of $members.values()) {
        if (matched) {
          return true;
        }
      }
      return false;
    }),
    empty: derived(members, ($members) => $members.size === 0),
    report(id, matched) {
      members.update((current) => {
        if (current.get(id) === matched) {
          return current;
        }
        const next = new Map(current);
        next.set(id, matched);
        return next;
      });
    },
    remove(id) {
      members.update((current) => {
        if (!current.has(id)) {
          return current;
        }
        const next = new Map(current);
        next.delete(id);
        return next;
      });
    },
  };
}

let nextMemberId = 0;

/** Unique id for one registry member, stable for the life of the component. */
export function nextMatchId(prefix: string): string {
  nextMemberId += 1;
  return `${prefix}-${nextMemberId}`;
}

export function provideSettingsQuery(query: Readable<string>): void {
  setContext(QUERY_KEY, query);
}

export function useSettingsQuery(): Readable<string> {
  return hasContext(QUERY_KEY) ? getContext<Readable<string>>(QUERY_KEY) : EMPTY_QUERY;
}

/** The card title is folded into every row's haystack, so "theme" reveals the whole card. */
export function provideCardTitle(title: Readable<string>): void {
  setContext(CARD_TITLE_KEY, title);
}

export function useCardTitle(): Readable<string> {
  return hasContext(CARD_TITLE_KEY) ? getContext<Readable<string>>(CARD_TITLE_KEY) : EMPTY_QUERY;
}

export function provideRowRegistry(registry: MatchRegistry): void {
  setContext(ROW_REGISTRY_KEY, registry);
}

export function useRowRegistry(): MatchRegistry | null {
  return hasContext(ROW_REGISTRY_KEY) ? getContext<MatchRegistry>(ROW_REGISTRY_KEY) : null;
}

export function provideCardRegistry(registry: MatchRegistry): void {
  setContext(CARD_REGISTRY_KEY, registry);
}

export function useCardRegistry(): MatchRegistry | null {
  return hasContext(CARD_REGISTRY_KEY) ? getContext<MatchRegistry>(CARD_REGISTRY_KEY) : null;
}
