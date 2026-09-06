// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const ROUTE_PARAM_KEYS: Record<string, string[]> = {
  "/library/:libraryId": ["libraryId"],
  "/item/:itemId": ["itemId"],
  "/person/:personId": ["personId"],
  "/play/:itemId": ["itemId"],
  "/music/album/:albumId": ["albumId"],
  "/music/artist/:artistId": ["artistId"],
  "/music/mix/:mixId": ["mixId"],
  "/music/playlist/:playlistId": ["playlistId"],
  "/music/server-playlist/:playlistId": ["playlistId"],
  "/music/genre/:genre": ["genre"],
  "/settings/:tab": ["tab"],
  "/share/:token": ["token"],
  "/listen/:token": ["token"],
};

export function getRouteComponentProps(
  path: string,
  params: Record<string, string>,
  query?: Record<string, string>,
): Record<string, string> {
  switch (path) {
    case "/library/:libraryId":
      return { libraryId: params.libraryId };
    case "/item/:itemId":
      return {
        itemId: params.itemId,
        ...(query?.season ? { seasonQuery: query.season } : {}),
      };
    case "/person/:personId":
      return { personId: params.personId };
    case "/play/:itemId":
      return { itemId: params.itemId };
    case "/music/album/:albumId":
      return { albumId: params.albumId };
    case "/music/artist/:artistId":
      return { artistId: params.artistId };
    case "/music/mix/:mixId":
      return { mixId: params.mixId };
    case "/music/playlist/:playlistId":
      return { playlistId: params.playlistId };
    case "/music/server-playlist/:playlistId":
      return { playlistId: params.playlistId };
    case "/music/genre/:genre":
      return { genre: params.genre };
    case "/settings/:tab":
      return { tab: params.tab };
    case "/share/:token":
      return { token: params.token };
    case "/listen/:token":
      return { token: params.token };
    default:
      return {};
  }
}

export function routePropsFromMatch(
  path: string,
  params: Record<string, string>,
  query?: Record<string, string>,
): Record<string, string> {
  return getRouteComponentProps(path, params, query);
}

export function requiredRouteParamKeys(path: string): string[] {
  return ROUTE_PARAM_KEYS[path] ?? [];
}

export function routesWithIncompleteProps(
  path: string,
  params: Record<string, string>,
): string[] {
  return requiredRouteParamKeys(path).filter((key) => !params[key]);
}

export function routesMissingParamMappings(routes: string[]): string[] {
  return routes.filter((path) => requiredRouteParamKeys(path).length === 0);
}
