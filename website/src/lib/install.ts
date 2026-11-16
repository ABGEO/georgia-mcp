/** The install link of Cursor for a remote server. */
export function cursorInstallUrl(name: string, url: string): string {
  const config = encodeURIComponent(btoa(JSON.stringify({ url })))

  return `cursor://anysphere.cursor-deeplink/mcp/install?name=${encodeURIComponent(name)}&config=${config}`
}
