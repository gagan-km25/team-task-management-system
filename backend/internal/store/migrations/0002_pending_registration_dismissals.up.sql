-- Pending-registration dismissals, per workspace.
-- Records which registered (workspace-less) users an administrator has
-- dismissed from their pending-registrations list, so a dismissed user stays
-- hidden for that workspace only. Accounts and other workspaces are unaffected.

CREATE TABLE IF NOT EXISTS dismissed_registrations (
  workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dismissed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_dismissed_registrations_workspace ON dismissed_registrations(workspace_id);
