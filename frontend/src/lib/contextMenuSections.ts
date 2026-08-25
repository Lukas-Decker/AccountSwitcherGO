/**
 * Which section title owns each row of a menu column.
 *
 * A title is a row like any other, so what belongs under it has to be worked out from the order
 * of the list: everything after a header, up to the next header, subheader or separator. A
 * separator ends a section because that is already how menus mark the end of a group, and
 * without that rule the single title at the top of a menu would own the whole menu and folding
 * it would empty the panel.
 */

export type SectionItem = { type?: string };

/**
 * For each row, the indices of the titles that fold it away, outermost first. Empty for rows
 * that stand outside any section, and for the titles themselves at their own level.
 */
export function menuSectionOwners(items: SectionItem[]): number[][] {
  const owners: number[][] = [];
  let header: number | null = null;
  let subheader: number | null = null;

  items.forEach((item, index) => {
    if (item.type === "separator") {
      header = null;
      subheader = null;
      owners.push([]);
      return;
    }
    if (item.type === "header") {
      owners.push([]);
      header = index;
      subheader = null;
      return;
    }
    if (item.type === "subheader") {
      /* A subheader disappears with the header above it, and takes its own rows with it. */
      owners.push(header === null ? [] : [header]);
      subheader = index;
      return;
    }
    const own: number[] = [];
    if (header !== null) {
      own.push(header);
    }
    if (subheader !== null) {
      own.push(subheader);
    }
    owners.push(own);
  });

  return owners;
}

/** True when folding the title at `index` would actually hide something. */
export function sectionHasRows(owners: number[][], index: number): boolean {
  return owners.some((own) => own.includes(index));
}
