<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import Spinner from '$lib/components/Spinner.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import {
    UserCog, Plus, Trash2, Pencil, X, Save,
    Search, MoreVertical, Shield, User, Crown
  } from 'lucide-svelte';

  interface UserRecord {
    id: string;
    name: string;
    email: string;
    role: string;
    status?: string;
    created_at: string;
    last_active?: string;
  }

  let users = $state<UserRecord[]>([]);
  let loading = $state(true);
  let error = $state('');
  let searchQuery = $state('');
  let saving = $state(false);
  let editingId = $state<string | null>(null);
  let showCreateForm = $state(false);

  // Form state
  let formName = $state('');
  let formEmail = $state('');
  let formRole = $state('member');
  let formStatus = $state('active');

  const roleColors: Record<string, string> = {
    owner: 'var(--color-warning)',
    admin: 'var(--color-purple)',
    member: 'var(--color-fg-2)',
  };

  async function loadUsers() {
    try {
      const params = new URLSearchParams();
      if (searchQuery.trim()) params.set('search', searchQuery.trim());
      const query = params.toString();
      const res = await api.get<{ data: UserRecord[] }>(`/api/users${query ? '?' + query : ''}`);
      const raw = res?.data;
      const arr = Array.isArray(raw) ? raw : [];
      // Normalize mixed data (some items have 'name', others 'username')
      users = arr.filter(u => u.name || u.username).map(u => ({
        ...u,
        name: u.name || u.username || 'Unknown',
        email: u.email || '',
        role: u.role || 'viewer',
        status: u.status || (u.active ? 'active' : 'inactive'),
        created_at: u.created_at || ''
      }));
    } catch (e: any) {
      error = e.message || 'Failed to load users';
    }
  }

  onMount(async () => {
    loading = true;
    await loadUsers();
    loading = false;
  });

  function resetForm() {
    formName = '';
    formEmail = '';
    formRole = 'member';
    formStatus = 'active';
    editingId = null;
    showCreateForm = false;
  }

  function startEdit(user: UserRecord) {
    editingId = user.id;
    formName = user.name;
    formEmail = user.email;
    formRole = user.role;
    formStatus = user.status;
    showCreateForm = false;
  }

  async function createUser() {
    if (!formName.trim() || !formEmail.trim()) return;
    saving = true;
    try {
      const res = await api.post<{ data: UserRecord }>('/api/users', {
        name: formName.trim(),
        email: formEmail.trim(),
        role: formRole,
        status: formStatus
      });
      const user = res.data || res as any;
      users = [...users, user];
      resetForm();
    } catch (e: any) {
      error = e.message || 'Failed to create user';
    }
    saving = false;
  }

  async function updateUser() {
    if (!editingId || !formName.trim() || !formEmail.trim()) return;
    saving = true;
    try {
      await api.put(`/api/users/${editingId}`, {
        name: formName.trim(),
        email: formEmail.trim(),
        role: formRole,
        status: formStatus
      });
      users = users.map(u => u.id === editingId ? {
        ...u, name: formName.trim(), email: formEmail.trim(), role: formRole, status: formStatus
      } : u);
      resetForm();
    } catch (e: any) {
      error = e.message || 'Failed to update user';
    }
    saving = false;
  }

  async function deleteUser(userId: string) {
    if (!confirm('Are you sure you want to delete this user?')) return;
    try {
      await api.delete(`/api/users/${userId}`);
      users = users.filter(u => u.id !== userId);
    } catch (e: any) {
      error = e.message || 'Failed to delete user';
    }
  }

  async function toggleStatus(user: UserRecord) {
    const newStatus = user.status === 'active' ? 'disabled' : 'active';
    try {
      await api.patch(`/api/users/${user.id}`, { status: newStatus });
      users = users.map(u => u.id === user.id ? { ...u, status: newStatus } : u);
    } catch (e: any) {
      error = e.message || 'Failed to update user status';
    }
  }

  function handleSearch() {
    loading = true;
    loadUsers().then(() => loading = false);
  }

  let isEditing = $derived(editingId !== null);
  let filteredUsers = $derived(users);

  function formatTime(ts: string): string {
    try {
      const diff = Date.now() - new Date(ts).getTime();
      const mins = Math.floor(diff / 60000);
      if (mins < 1) return 'just now';
      if (mins < 60) return `${mins}m ago`;
      const hrs = Math.floor(mins / 60);
      if (hrs < 24) return `${hrs}h ago`;
      return `${Math.floor(hrs / 24)}d ago`;
    } catch {
      return ts;
    }
  }
</script>

<svelte:head>
  <title>Users — Lintasan</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 24px;">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2.5">
      <div
        class="flex items-center justify-center rounded-xl"
        style="width: 40px; height: 40px; background: var(--color-info-light);"
      >
        <UserCog size={20} style="color: var(--color-info);" stroke-width={1.8} />
      </div>
      <div>
        <div style="font-size: 15px; font-weight: 600; color: var(--color-fg-0);">Users</div>
        <div style="font-size: 12px; color: var(--color-fg-3);">Manage user accounts and permissions</div>
      </div>
    </div>
    <button
      class="btn-primary flex items-center gap-1.5"
      onclick={() => { resetForm(); showCreateForm = !showCreateForm; }}
    >
      <Plus size={14} stroke-width={2} />
      Add User
    </button>
  </div>

  <!-- Search Bar -->
  <div class="card" style="padding: 16px 20px;">
    <div class="flex items-center gap-3">
      <div class="search-wrapper">
        <Search size={14} style="color: var(--color-fg-3); position: absolute; left: 10px; top: 50%; transform: translateY(-50%); pointer-events: none;" />
        <input
          class="input-field search-input"
          placeholder="Search users by name or email..."
          bind:value={searchQuery}
          onkeydown={(e) => e.key === 'Enter' && handleSearch()}
        />
      </div>
      <button class="btn-primary" onclick={handleSearch} style="padding: 7px 14px;">Search</button>
    </div>
  </div>

  <!-- Create/Edit Form -->
  {#if showCreateForm || isEditing}
    <div class="card" style="animation: fadeInUp 0.3s ease-out;">
      <div class="flex items-center justify-between" style="margin-bottom: 16px;">
        <div style="font-size: 14px; font-weight: 600; color: var(--color-fg-0);">
          {isEditing ? 'Edit User' : 'New User'}
        </div>
        <button class="btn-icon" onclick={resetForm}>
          <X size={16} style="color: var(--color-fg-3);" />
        </button>
      </div>
      <div class="grid gap-3" style="grid-template-columns: 1fr 1fr 150px 130px;">
        <input class="input-field" placeholder="Full name" bind:value={formName} />
        <input class="input-field" placeholder="Email address" bind:value={formEmail} />
        <select class="input-field" bind:value={formRole}>
          <option value="member">Member</option>
          <option value="admin">Admin</option>
          <option value="owner">Owner</option>
        </select>
        <select class="input-field" bind:value={formStatus}>
          <option value="active">Active</option>
          <option value="disabled">Disabled</option>
          <option value="pending">Pending</option>
        </select>
      </div>
      <div class="flex items-center gap-2" style="margin-top: 16px;">
        <button
          class="btn-primary flex items-center gap-1.5"
          onclick={isEditing ? updateUser : createUser}
          disabled={saving || !formName.trim() || !formEmail.trim()}
        >
          <Save size={14} />
          {saving ? 'Saving...' : (isEditing ? 'Update User' : 'Create User')}
        </button>
        <button class="btn-secondary" onclick={resetForm}>Cancel</button>
      </div>
    </div>
  {/if}

  <!-- Users Table -->
  <div class="card" style="padding: 0; overflow: hidden;">
    {#if loading}
      <Spinner />
    {:else if users.length === 0}
      <EmptyState
        icon={UserCog}
        title="No users found"
        description={searchQuery.trim() ? 'Try adjusting your search.' : 'Add users to grant dashboard access.'}
      />
    {:else}
      <div style="overflow-x: auto;">
        <table class="users-table">
          <thead>
            <tr>
              <th style="width: 200px;">Name</th>
              <th style="width: 240px;">Email</th>
              <th style="width: 110px;">Role</th>
              <th style="width: 100px;">Status</th>
              <th style="width: 140px;">Last Active</th>
              <th style="width: 120px;">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredUsers as user, i (user.id)}
              <tr style="animation: fadeInUp {0.3 + i * 0.03}s ease-out;">
                <td>
                  <div class="flex items-center gap-2.5">
                    <div class="user-avatar">
                      {user.name.charAt(0).toUpperCase()}
                    </div>
                    <span style="font-size: 13px; font-weight: 500; color: var(--color-fg-0);">
                      {user.name}
                    </span>
                  </div>
                </td>
                <td>
                  <span class="font-mono" style="font-size: 12px; color: var(--color-fg-2);">
                    {user.email}
                  </span>
                </td>
                <td>
                  <span
                    class="badge flex items-center gap-1"
                    style="font-size: 10px; display: inline-flex; background: {roleColors[user.role] || 'var(--color-fg-2)'}15; color: {roleColors[user.role] || 'var(--color-fg-2)'};"
                  >
                    {user.role}
                  </span>
                </td>
                <td>
                  <StatusBadge status={user.status} />
                </td>
                <td>
                  <span style="font-size: 12px; color: var(--color-fg-3);">
                    {user.last_active ? formatTime(user.last_active) : '—'}
                  </span>
                </td>
                <td>
                  <div class="flex items-center gap-1">
                    <button
                      class="btn-icon"
                      style="color: var(--color-fg-2);"
                      onclick={() => startEdit(user)}
                      title="Edit user"
                    >
                      <Pencil size={14} />
                    </button>
                    <button
                      class="btn-icon"
                      style="color: {user.status === 'active' ? 'var(--color-warning)' : 'var(--color-success)'};"
                      onclick={() => toggleStatus(user)}
                      title={user.status === 'active' ? 'Disable user' : 'Enable user'}
                    >
                      <Shield size={14} />
                    </button>
                    <button
                      class="btn-icon"
                      style="color: var(--color-error);"
                      onclick={() => deleteUser(user.id)}
                      title="Delete user"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Table Footer -->
      <div
        class="flex items-center justify-between"
        style="padding: 12px 16px; border-top: 1px solid var(--color-border); background: var(--color-bg-body);"
      >
        <span style="font-size: 12px; color: var(--color-fg-3);">
          {users.length} user{users.length !== 1 ? 's' : ''}
        </span>
      </div>
    {/if}
  </div>

  {#if error}
    <div
      class="flex items-center gap-2"
      style="
        padding: 12px 16px; border-radius: var(--radius-sm);
        background: var(--color-error-light); color: var(--color-error);
        font-size: 13px; font-weight: 500;
      "
    >
      {error}
      <button style="margin-left: auto; cursor: pointer; color: var(--color-error); background: none; border: none;" onclick={() => error = ''}>&times;</button>
    </div>
  {/if}
</div>

<style>
  .search-wrapper {
    position: relative;
    flex: 1;
    min-width: 200px;
    max-width: 360px;
  }
  .search-input {
    padding-left: 32px !important;
  }
  .users-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  .users-table th {
    padding: 10px 16px;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--color-fg-3);
    background: var(--color-bg-body);
    border-bottom: 1px solid var(--color-border);
    white-space: nowrap;
  }
  .users-table td {
    padding: 12px 16px;
    border-bottom: 1px solid var(--color-border-light);
    vertical-align: middle;
  }
  .users-table tbody tr {
    transition: var(--transition);
  }
  .users-table tbody tr:hover {
    background: var(--color-primary-light);
  }
  .users-table tbody tr:last-child td {
    border-bottom: none;
  }
  .user-avatar {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: var(--color-primary-light);
    color: var(--color-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 600;
    flex-shrink: 0;
  }
  .btn-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    border: none;
    background: transparent;
    cursor: pointer;
    transition: var(--transition);
  }
  .btn-icon:hover {
    background: var(--color-bg-sidebar-hover);
  }
  @media (max-width: 768px) {
    .search-wrapper {
      max-width: 100%;
    }
  }
</style>
