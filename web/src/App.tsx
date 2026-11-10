import { useEffect, useState } from 'react'

export function App() {
  const [version, setVersion] = useState<string>()

  useEffect(() => {
    fetch('/api/version')
      .then((res) => res.json() as Promise<{ version: string }>)
      .then((body) => setVersion(body.version))
      .catch(() => setVersion('unknown'))
  }, [])

  return (
    <main>
      <h1>saas-test</h1>
      <p>Team subscriptions: plans, trials and seats. Coming soon.</p>
      <p data-testid="version">Version: {version ?? '…'}</p>
    </main>
  )
}
