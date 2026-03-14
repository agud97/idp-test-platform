import React, { useEffect, useState } from 'react';

type TeamStatus = {
  name: string;
  total: number;
  completed: number;
  validated: number;
  inProgress: number;
  pending: number;
  blocked: number;
  percentage: number;
};

type MigrationStatusResponse = {
  teams: TeamStatus[];
};

async function loadStatus(): Promise<MigrationStatusResponse> {
  const response = await fetch('/api/migration/status');
  if (!response.ok) {
    throw new Error(`status request failed: ${response.status}`);
  }
  return response.json();
}

export function MigrationDashboard(): JSX.Element {
  const [teams, setTeams] = useState<TeamStatus[]>([]);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    let cancelled = false;

    const refresh = async () => {
      try {
        const data = await loadStatus();
        if (!cancelled) {
          setTeams(data.teams);
          setError(undefined);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      }
    };

    refresh();
    const timer = window.setInterval(refresh, 60_000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  const gatePassed = teams.length > 0 && teams.every(team => team.total > 0 && team.completed === team.total);

  return (
    <section>
      <h1>Migration Dashboard</h1>
      {gatePassed ? <div>Gate PASSED</div> : null}
      {error ? <div>{error}</div> : null}
      <table>
        <thead>
          <tr>
            <th>Team</th>
            <th>Total</th>
            <th>Completed</th>
            <th>Validated</th>
            <th>In Progress</th>
            <th>Blocked</th>
            <th>%</th>
          </tr>
        </thead>
        <tbody>
          {teams.map(team => (
            <tr key={team.name}>
              <td>{team.name}</td>
              <td>{team.total}</td>
              <td>{team.completed}</td>
              <td>{team.validated}</td>
              <td>{team.inProgress}</td>
              <td>{team.blocked}</td>
              <td>{team.percentage}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

export default MigrationDashboard;
