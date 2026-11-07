/** The page of the release of a site version, or the list of releases when the version names no release. */
export function releaseUrl(repo: string, version: string): string {
  return /^v\d+\.\d+\.\d+$/.test(version) ? `${repo}/releases/tag/${version}` : `${repo}/releases`
}
