import { useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Plus, RefreshCw, ShieldCheck, Trash2 } from "lucide-react";
import {
  createSchoolRole,
  getCustomRolePermissions,
  getSchoolPermissionDelegations,
  getSchoolPermissions,
  getSchoolRoles,
  grantCustomRolePermission,
  revokeCustomRolePermission,
  type SchoolRole,
} from "../api";
import { Select } from "../../../shared/components/Select";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import "./RoleManagementPage.css";

type Props = { permissions: string[] };

export function RoleManagementPage({ permissions }: Props) {
  const queryClient = useQueryClient();
  const canCreate = permissions.includes("roles.create_custom");
  const canReadCatalogue = permissions.includes("permissions.read");
  const canManagePermissions = permissions.includes("permissions.assign") && permissions.includes("roles.update_custom");
  const [name, setName] = useState("");
  const [selectedRoleID, setSelectedRoleID] = useState<number | null>(null);
  const [permissionKey, setPermissionKey] = useState("");
  const [feedback, setFeedback] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [nameError, setNameError] = useState("");
  const [permissionToRevoke, setPermissionToRevoke] = useState<string | null>(null);
  const [grantDialogOpen, setGrantDialogOpen] = useState(false);
  const [grantFieldError, setGrantFieldError] = useState("");

  const rolesQuery = useQuery({
    queryKey: ["school-admin", "roles"],
    queryFn: ({ signal }) => getSchoolRoles(signal),
    staleTime: 30_000,
  });
  const catalogueQuery = useQuery({
    queryKey: ["school-admin", "permission-catalogue"],
    queryFn: ({ signal }) => getSchoolPermissions(signal),
    enabled: canReadCatalogue,
    staleTime: 60_000,
  });
  const delegationsQuery = useQuery({
    queryKey: ["school-admin", "permission-delegations"],
    queryFn: ({ signal }) => getSchoolPermissionDelegations(signal),
    enabled: canManagePermissions,
    staleTime: 15_000,
  });

  const roles = rolesQuery.data?.roles ?? [];
  const selectedRole = roles.find(role => role.id === selectedRoleID) ?? null;
  const rolePermissionsQuery = useQuery({
    queryKey: ["school-admin", "role-permissions", selectedRoleID],
    queryFn: ({ signal }) => getCustomRolePermissions(selectedRoleID!, signal),
    enabled: selectedRoleID !== null && selectedRole !== null && !selectedRole.is_system,
    staleTime: 15_000,
  });

  const create = useMutation({
    mutationFn: () => createSchoolRole(name.trim()),
    onSuccess: async result => {
      setName("");
      setCreateDialogOpen(false);
      setNameError("");
      setFeedback("Custom role created. Add only the permissions delegated to this school.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "roles"] });
      setSelectedRoleID(result.role.id);
    },
  });
  const grant = useMutation({
    mutationFn: (input: { permission_key: string; delegation_id: number }) =>
      grantCustomRolePermission(selectedRoleID!, input),
    onSuccess: async () => {
      setPermissionKey("");
      setGrantDialogOpen(false);
      setGrantFieldError("");
      setFeedback("Permission granted. It is effective only while its exact delegation remains active.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "role-permissions", selectedRoleID] });
    },
  });
  const revoke = useMutation({
    mutationFn: (key: string) => revokeCustomRolePermission(selectedRoleID!, key),
    onSuccess: async () => {
      setPermissionToRevoke(null);
      setFeedback("Permission grant removed.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "role-permissions", selectedRoleID] });
    },
  });

  const grants = rolePermissionsQuery.data?.permissions ?? [];
  const activeDelegations = useMemo(
    () => (delegationsQuery.data?.delegations ?? []).filter(item => item.can_assign_to_roles && !item.revoked_at),
    [delegationsQuery.data?.delegations],
  );
  const activeDelegationForSelection = activeDelegations.find(item => item.permission_key === permissionKey);
  const assignedKeys = new Set(grants.map(item => item.permission_key));
  const availableDelegations = activeDelegations.filter(item => !assignedKeys.has(item.permission_key));
  const catalogue = catalogueQuery.data?.permissions ?? [];
  const permissionDescription = (key: string) => catalogue.find(item => item.permission_key === key)?.description ?? "";

  function submitCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (name.trim().length < 2) { setNameError("Enter a role name with at least 2 characters."); return; }
    setNameError("");
    setFeedback("");
    create.mutate();
  }
  function submitGrant(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!permissionKey || !activeDelegationForSelection || !selectedRoleID) { setGrantFieldError("Choose a permission that has an active delegation for this school."); return; }
    setGrantFieldError("");
    setFeedback("");
    grant.mutate({
      permission_key: activeDelegationForSelection.permission_key,
      delegation_id: activeDelegationForSelection.id,
    });
  }

  return (
    <div className="content role-management-page">
      <div className="heading">
        <div>
          <span className="eyebrow">ACCESS CONTROL</span>
          <h1>Roles &amp; permissions</h1>
          <p>Create school-specific roles and manage permissions within the limits delegated by the Platform Admin.</p>
        </div>
        <div className="role-page-count"><ShieldCheck size={16}/>{roles.length} roles</div>
      </div>

      {canCreate && <section className="role-create-panel"><div><h2>Create a custom role</h2><p>Use a clear name that describes the staff responsibility.</p></div><button type="button" className="primary" onClick={() => {create.reset();setFeedback("");setName("");setNameError("");setCreateDialogOpen(true);}}><Plus size={15}/> Create role</button></section>}

      <section className="role-list-panel">
        <div className="role-section-heading"><div><h2>School roles</h2><p>Select a custom role to review its permission grants.</p></div><button type="button" className="role-refresh" onClick={() => void rolesQuery.refetch()} disabled={rolesQuery.isFetching} aria-label="Refresh roles"><RefreshCw size={15}/></button></div>
        {rolesQuery.isPending ? <div className="role-state">Loading school roles…</div>
          : rolesQuery.isError ? <div className="role-state role-error">{rolesQuery.error instanceof Error ? rolesQuery.error.message : "Unable to load school roles."}<button type="button" onClick={() => void rolesQuery.refetch()}>Try again</button></div>
          : roles.length === 0 ? <div className="role-state">No roles are available for this school.</div>
          : <div className="role-list">{roles.map((role: SchoolRole) => (
            <article className={selectedRoleID === role.id ? "role-card selected" : "role-card"} key={role.id}>
              <div className="role-card-main">
                <div className="role-card-icon"><ShieldCheck size={17}/></div>
                <div className="role-card-copy"><b>{role.name}</b><small>{role.role_key}</small></div>
                <span className={role.is_system ? "role-kind system" : "role-kind custom"}>{role.is_system ? "System" : "Custom"}</span>
              </div>
              {!role.is_system && <button type="button" className="role-manage-button" onClick={() => { setSelectedRoleID(current => current === role.id ? null : role.id); setFeedback(""); setPermissionKey(""); }}>
                {selectedRoleID === role.id ? "Close permissions" : "Manage permissions"}
              </button>}
              {role.is_system && <p className="role-system-note">Protected built-in role · permissions are managed by the platform.</p>}
            </article>
          ))}</div>}
      </section>

      {selectedRole && !selectedRole.is_system && (
        <Modal open title={`Manage permissions · ${selectedRole.name}`} description="Review and update this custom role’s delegated permissions. Changes are checked against the Platform Admin’s active delegation." onClose={() => {setSelectedRoleID(null);setPermissionKey("");setFeedback("");}}>
        <section className="role-permission-panel">
          <div className="role-section-heading"><div><span className="eyebrow">CUSTOM ROLE</span><h2>{selectedRole.name}</h2><p>Current permission grants and their exact delegation status.</p></div></div>
          {rolePermissionsQuery.isPending ? <div className="role-state">Loading permissions…</div>
            : rolePermissionsQuery.isError ? <div className="role-state role-error">{rolePermissionsQuery.error instanceof Error ? rolePermissionsQuery.error.message : "Unable to load role permissions."}<button type="button" onClick={() => void rolePermissionsQuery.refetch()}>Try again</button></div>
            : grants.length === 0 ? <div className="role-state">No permissions have been granted to this role yet.</div>
            : <ul className="role-grant-list">{grants.map(item => {
              const effective = item.delegation_active && item.permission_active;
              return <li key={item.permission_key}>
                <div className="role-grant-copy"><b>{item.permission_key}</b>{item.description && <small>{item.description}</small>}<span className={effective ? "role-grant-status active" : "role-grant-status inactive"}>{effective ? "Active grant" : "Inactive · delegation revoked or permission disabled"}</span></div>
                {canManagePermissions && <button type="button" className="role-remove-button" disabled={revoke.isPending} onClick={() => {revoke.reset();setPermissionToRevoke(item.permission_key);}} aria-label={`Remove ${item.permission_key}`}><Trash2 size={15}/></button>}
              </li>;
            })}</ul>}

          {canManagePermissions ? (
            <div className="role-grant-form">
              <p>Only active permissions explicitly delegated by the Platform Admin can be assigned to this role.</p>
              <button type="button" className="primary" onClick={() => {grant.reset();setFeedback("");setPermissionKey("");setGrantFieldError("");setGrantDialogOpen(true);}} disabled={availableDelegations.length===0 || delegationsQuery.isLoading}><Plus size={15}/> Add delegated permission</button>
              {!delegationsQuery.isPending && !delegationsQuery.isError && availableDelegations.length===0 && <small>No unused permissions have an active Platform Admin delegation for this school.</small>}
            </div>
          ) : <p className="role-readonly-note"><AlertTriangle size={15}/> You can view permissions but do not have both permissions required to change this role.</p>}

          {(feedback || ((!grantDialogOpen && permissionToRevoke === null) && (grant.isError || revoke.isError))) && <p className={grant.isError || revoke.isError ? "role-feedback role-error" : "role-feedback"} role="status">
            {grant.isError ? (grant.error instanceof Error ? grant.error.message : "Unable to grant permission.")
              : revoke.isError ? (revoke.error instanceof Error ? revoke.error.message : "Unable to remove permission.")
              : feedback}
          </p>}
        </section>
        </Modal>
      )}
      <Modal open={createDialogOpen} title="Create a custom role" description="Choose a clear name that describes the staff responsibility. Custom roles belong only to this school." onClose={() => setCreateDialogOpen(false)} closeDisabled={create.isPending}>
        {create.isError && <p className="role-error" role="alert">{create.error instanceof Error ? create.error.message : "Unable to create this role. Check the details and try again."}</p>}
        <form onSubmit={submitCreate} noValidate>
          <label htmlFor="custom-role-name">Role name</label>
          <input id="custom-role-name" value={name} onChange={event => {setName(event.target.value);if(event.target.value.trim().length>=2)setNameError("");}} minLength={2} maxLength={120} placeholder="e.g. Library Coordinator" required aria-invalid={Boolean(nameError)} aria-describedby={nameError ? "custom-role-name-error" : "custom-role-name-help"}/>
          <small id="custom-role-name-help">Use a name that clearly explains what this staff role is responsible for.</small>
          {nameError && <small className="app-field-error" id="custom-role-name-error">{nameError}</small>}
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => setCreateDialogOpen(false)} disabled={create.isPending}>Cancel</button><button className="primary" type="submit" disabled={create.isPending}>{create.isPending?"Creating…":"Create role"}</button></div>
        </form>
      </Modal>
      <Modal open={grantDialogOpen} title="Grant a delegated permission" description={selectedRole ? `Choose a permission to add to the ${selectedRole.name} role. Only active delegations that allow role assignment are available.` : ""} onClose={()=>setGrantDialogOpen(false)} closeDisabled={grant.isPending}>
        {grant.isError&&<p className="role-error" role="alert">{grant.error instanceof Error?grant.error.message:"Unable to grant this permission. Please try again."}</p>}
        <form className="role-grant-form" onSubmit={submitGrant} noValidate>
          <label htmlFor="role-permission-select">Permission</label>
          <Select ariaLabel="Permission to grant" invalid={Boolean(grantFieldError)} describedBy={grantFieldError?"role-grant-error":undefined} value={permissionKey} placeholder="Choose an available permission" onChange={value=>{setPermissionKey(value);setGrantFieldError("");}} disabled={grant.isPending||availableDelegations.length===0} options={availableDelegations.map(item=>({value:item.permission_key,label:item.permission_key+(permissionDescription(item.permission_key)?` — ${permissionDescription(item.permission_key)}`:"")}))}/>
          {grantFieldError&&<small className="app-field-error" id="role-grant-error" role="alert">{grantFieldError}</small>}
          {delegationsQuery.isPending&&<small>Loading Platform Admin delegations…</small>}
          {delegationsQuery.isError&&<small className="role-error">{delegationsQuery.error instanceof Error?delegationsQuery.error.message:"Unable to load active delegations."}</small>}
          <small>The backend revalidates the exact delegation before saving.</small>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={()=>setGrantDialogOpen(false)} disabled={grant.isPending}>Cancel</button><button className="primary" type="submit" disabled={grant.isPending||availableDelegations.length===0}>{grant.isPending?"Granting…":"Grant permission"}</button></div>
        </form>
      </Modal>
      <ConfirmDialog open={permissionToRevoke !== null} title="Remove permission grant?" description={permissionToRevoke ? `Remove ${permissionToRevoke} from ${selectedRole?.name ?? "this role"}? The role will no longer receive this permission through this grant.` : ""} confirmLabel="Remove permission" pending={revoke.isPending} error={revoke.isError?(revoke.error instanceof Error?revoke.error.message:"Unable to remove this permission. Please try again."):undefined} onCancel={() => setPermissionToRevoke(null)} onConfirm={() => { if(permissionToRevoke){setFeedback("");revoke.mutate(permissionToRevoke);} }}/>
    </div>
  );
}
