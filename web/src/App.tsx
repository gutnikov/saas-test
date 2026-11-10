import { type FormEvent, useEffect, useState } from 'react'

type Team = { id: string; name: string }

// TeamsPage (spec TEAMS-1): the form that creates a team, and the list of teams, newest first.
export function App() {
  const [teams, setTeams] = useState<Team[]>([])
  const [name, setName] = useState('')
  const [error, setError] = useState<string>()
  const [version, setVersion] = useState<string>()

  async function loadTeams() {
    const res = await fetch('/api/teams')
    if (res.ok) setTeams((await res.json()) as Team[])
  }

  useEffect(() => {
    void loadTeams()
    fetch('/api/version')
      .then((res) => res.json() as Promise<{ version: string }>)
      .then((body) => setVersion(body.version))
      .catch(() => setVersion('unknown'))
  }, [])

  async function createTeam(event: FormEvent) {
    event.preventDefault()
    const res = await fetch('/api/teams', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    const body = (await res.json()) as { message?: string }
    if (!res.ok) {
      setError(body.message ?? 'Something went wrong')
      return
    }
    setName('')
    setError(undefined)
    await loadTeams()
  }

  return (
    <main>
      <h1>saas-test</h1>
      <form onSubmit={createTeam}>
        <label>
          Team name{' '}
          <input value={name} onChange={(e) => setName(e.target.value)} data-testid="team-name" />
        </label>{' '}
        <button type="submit">Create team</button>
        {error && (
          <p role="alert" data-testid="team-error">
            {error}
          </p>
        )}
      </form>
      <ul data-testid="teams">
        {teams.map((team) => (
          <li key={team.id}>{team.name}</li>
        ))}
      </ul>
      <p data-testid="version">Version: {version ?? '…'}</p>
    </main>
  )
}
