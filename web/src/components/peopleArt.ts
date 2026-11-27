/**
 * Named portraits: `src/assets/art/people/<card id>.jpg`, generated from photos of club members
 * (published with the user's OK). Cards without a portrait fall back to the generic art.
 */
const peopleFiles = import.meta.glob<string>("../assets/art/people/*.{webp,png,jpg}", {
  eager: true,
  query: "?url",
  import: "default",
});

export const PEOPLE_MANIFEST: Record<string, string> = Object.fromEntries(
  Object.entries(peopleFiles).map(([path, url]) => [path.replace(/^.*\/(.+)\.\w+$/, "$1"), url]),
);
