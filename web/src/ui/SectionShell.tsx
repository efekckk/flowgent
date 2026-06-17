import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../auth/useAuth';
import LogSearchBar from '../search/LogSearchBar';
import Icon from './Icon';

type NavItem = {
  to: string;
  label: string;
  icon?: ReactNode;
  active?: boolean;
};

interface SidebarProps {
  workflowId?: string | null;
  workflowName?: string | null;
  active: 'editor' | 'triggers' | 'runs' | 'credentials';
}

export function SectionSidebar({ workflowId, workflowName, active }: SidebarProps) {
  const { logout, workspace, user } = useAuth();

  const items: NavItem[] = [];
  if (workflowId) {
    items.push({ to: `/workflows/${workflowId}`,          label: 'editor',   icon: <Icon name="braces" size={12} />,   active: active === 'editor' });
    items.push({ to: `/workflows/${workflowId}/triggers`, label: 'triggers', icon: <Icon name="clock" size={12} />,    active: active === 'triggers' });
    items.push({ to: `/workflows/${workflowId}/runs`,     label: 'runs',     icon: <Icon name="scroll" size={12} />,   active: active === 'runs' });
  }
  items.push({ to: '/credentials', label: 'credentials', icon: <Icon name="key" size={12} />, active: active === 'credentials' });

  return (
    <aside className="flex h-full w-64 flex-col border-r border-ink-500 bg-ink-700">
      <div className="border-b border-ink-500 px-5 py-4">
        <Link to="/workflows" className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.28em] text-paper-400 hover:text-cyan">
          <Icon name="arrow-left" size={12} />
          drafting room
        </Link>
        {workflowName && (
          <div className="mt-3">
            <div className="font-mono text-[10px] uppercase tracking-[0.32em] text-paper-600">sheet</div>
            <div className="h-display mt-0.5 truncate text-lg text-paper-50">{workflowName}</div>
            {workflowId && (
              <div className="mt-0.5 truncate font-mono text-[10px] uppercase tracking-[0.22em] text-paper-600">
                {workflowId}
              </div>
            )}
          </div>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto py-3">
        <div className="px-5 pb-2 font-mono text-[10px] uppercase tracking-[0.32em] text-paper-400">
          views
        </div>
        <ul className="space-y-px">
          {items.map((it) => (
            <li key={it.to}>
              <Link
                to={it.to}
                className={`flex items-center gap-3 border-l-2 px-5 py-2 font-mono text-xs uppercase tracking-[0.24em] transition ${
                  it.active
                    ? 'border-cyan bg-cyan/5 text-cyan'
                    : 'border-transparent text-paper-200 hover:border-ink-400 hover:bg-ink-600/60 hover:text-paper-50'
                }`}
              >
                <span className={it.active ? 'text-cyan' : 'text-paper-400'}>{it.icon}</span>
                <span>{it.label}</span>
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      <div className="border-t border-ink-500 px-5 py-4 space-y-3">
        <LogSearchBar workspaceId={workspace?.id ?? null} />
        <div className="font-mono text-[10px] uppercase tracking-[0.28em] text-paper-600">
          <div className="truncate text-paper-400">{user?.email}</div>
          {workspace && <div className="truncate">ws · {workspace.name}</div>}
        </div>
        <button
          onClick={logout}
          className="flex w-full items-center gap-2 rounded-sharp px-1 py-1.5 font-mono text-[10px] uppercase tracking-[0.24em] text-paper-600 transition hover:text-paper-200"
        >
          <Icon name="logout" size={12} />
          sign out
        </button>
      </div>
    </aside>
  );
}

interface PageHeaderProps {
  eyebrow: string;
  title: string;
  meta?: ReactNode;
  actions?: ReactNode;
  description?: string;
}

export function PageHeader({ eyebrow, title, meta, actions, description }: PageHeaderProps) {
  return (
    <div className="border-b border-ink-500 bg-ink-700/60 px-8 py-6">
      <div className="flex items-start justify-between gap-6">
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.36em] text-cyan">{eyebrow}</div>
          <h1 className="h-display mt-2 text-3xl text-paper-50">{title}</h1>
          {description && <p className="mt-2 max-w-xl font-mono text-xs text-paper-400">{description}</p>}
          {meta && <div className="mt-3 flex flex-wrap items-center gap-2">{meta}</div>}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

interface StatusPillProps {
  status: string;
  className?: string;
}

export function StatusPill({ status, className = '' }: StatusPillProps) {
  let tone = 'pill';
  switch (status) {
    case 'succeeded': tone = 'pill pill-moss'; break;
    case 'failed':    tone = 'pill pill-rose'; break;
    case 'running':   tone = 'pill pill-amber'; break;
    case 'cancelled': tone = 'pill'; break;
    case 'pending':   tone = 'pill pill-cyan'; break;
    case 'armed':     tone = 'pill pill-cyan'; break;
    default:          tone = 'pill';
  }
  return (
    <span className={`${tone} ${className}`}>
      <Icon name="dot" size={6} />
      {status}
    </span>
  );
}
