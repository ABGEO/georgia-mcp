/** The version of a release without its "v", or undefined when the site version names no release. */
export function releaseVersion(version: string): string | undefined {
  return /^v\d+\.\d+\.\d+$/.test(version) ? version.slice(1) : undefined
}

/** The page of the release of a site version, or the list of releases when the version names no release. */
export function releaseUrl(repo: string, version: string): string {
  return releaseVersion(version) ? `${repo}/releases/tag/${version}` : `${repo}/releases`
}
