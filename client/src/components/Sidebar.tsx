import React from 'react';
import { LayoutDashboard, Key, Sliders, BarChart3, FileJson, Terminal, ShieldAlert, Layers } from 'lucide-react';

export type TabId = 'overview' | 'tenant-catalog' | 'api-keys' | 'metering' | 'analytics' | 'json-studio' | 'api-tester';

interface SidebarProps {
  activeTab: TabId;
  onTabChange: (tab: TabId) => void;
}

const navItems: Array<{ id: TabId; label: string; icon: React.FC<{ className?: string }>; badge?: string }> = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'tenant-catalog', label: 'Tenant Catalog', icon: Layers, badge: 'MaaS' },
  { id: 'api-keys', label: 'API Keys', icon: Key },
  { id: 'metering', label: 'Metering Sandbox', icon: Sliders, badge: 'Sliders' },
  { id: 'analytics', label: 'Usage & Storage', icon: BarChart3 },
  { id: 'json-studio', label: 'JSON Studio', icon: FileJson, badge: 'JSON' },
  { id: 'api-tester', label: 'API Playground', icon: Terminal },
];

export const Sidebar: React.FC<SidebarProps> = ({ activeTab, onTabChange }) => {
  return (
    <aside className="w-full md:w-64 bg-obsidian-900/80 border-r border-gold-500/15 p-4 flex flex-col justify-between shrink-0">
      <div className="space-y-6">
        <div className="px-3 py-2">
          <p className="text-[10px] font-semibold uppercase tracking-wider text-gold-500/80">
            Navigation Menu
          </p>
        </div>

        <nav className="space-y-1.5">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeTab === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onTabChange(item.id)}
                className={`w-full flex items-center justify-between px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 ${
                  isActive
                    ? 'bg-gradient-to-r from-gold-500/20 to-gold-700/10 text-gold-300 border border-gold-500/40 shadow-gold-sm'
                    : 'text-slate-400 hover:text-slate-100 hover:bg-obsidian-800/60'
                }`}
              >
                <div className="flex items-center space-x-3">
                  <Icon className={`w-4 h-4 ${isActive ? 'text-gold-400' : 'text-slate-400'}`} />
                  <span>{item.label}</span>
                </div>
                {item.badge && (
                  <span
                    className={`text-[10px] px-1.5 py-0.5 rounded font-mono font-semibold ${
                      isActive
                        ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40'
                        : 'bg-obsidian-800 text-slate-400 border border-slate-700'
                    }`}
                  >
                    {item.badge}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Commercial VIP Card Footer */}
      <div className="mt-8 p-4 rounded-xl glass-panel-gold space-y-2">
        <div className="flex items-center space-x-2">
          <ShieldAlert className="w-4 h-4 text-gold-400 shrink-0" />
          <h4 className="text-xs font-semibold text-gold-300">Production Mode</h4>
        </div>
        <p className="text-[11px] text-slate-400 leading-relaxed">
          Tested for MVP deployment with direct REST JSON endpoint fallback.
        </p>
      </div>
    </aside>
  );
};
