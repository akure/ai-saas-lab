import React from 'react';
import { Calendar, Search, Filter, X } from 'lucide-react';
import { CatalogDateFilterState } from '../../types/catalog';

interface CatalogDateFilterProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  dateFilter: CatalogDateFilterState;
  onDateFilterChange: (df: CatalogDateFilterState) => void;
  entityTypeFilter: 'all' | 'services' | 'metrics' | 'plans';
  onEntityTypeFilterChange: (t: 'all' | 'services' | 'metrics' | 'plans') => void;
}

export const CatalogDateFilter: React.FC<CatalogDateFilterProps> = ({
  searchQuery,
  onSearchChange,
  dateFilter,
  onDateFilterChange,
  entityTypeFilter,
  onEntityTypeFilterChange,
}) => {
  return (
    <div className="p-4 rounded-xl glass-panel flex flex-col md:flex-row md:items-center justify-between gap-4">
      {/* Search Input */}
      <div className="relative flex-1 min-w-[240px]">
        <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search by ID, name, unit, or description..."
          className="w-full pl-10 pr-9 py-2 bg-obsidian-900 border border-gold-500/20 rounded-lg text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-gold-500/60 transition-all font-sans"
        />
        {searchQuery && (
          <button
            onClick={() => onSearchChange('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      {/* Entity Filter Pills */}
      <div className="flex items-center space-x-1.5 bg-obsidian-900 p-1 rounded-lg border border-gold-500/20 self-start md:self-auto overflow-x-auto">
        {(
          [
            { id: 'all', label: 'All Catalog' },
            { id: 'services', label: 'Services' },
            { id: 'metrics', label: 'Metrics' },
            { id: 'plans', label: 'Plans' },
          ] as const
        ).map((item) => (
          <button
            key={item.id}
            onClick={() => onEntityTypeFilterChange(item.id)}
            className={`px-3 py-1 text-xs font-semibold rounded-md transition-all whitespace-nowrap ${
              entityTypeFilter === item.id
                ? 'bg-gold-500/20 text-gold-300 border border-gold-500/40 shadow-gold-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>

      {/* Date Range Selector */}
      <div className="flex items-center space-x-2 self-start md:self-auto">
        <div className="flex items-center space-x-1.5 px-2.5 py-1.5 bg-obsidian-900 border border-gold-500/20 rounded-lg text-xs text-slate-300">
          <Calendar className="w-3.5 h-3.5 text-gold-400" />
          <select
            value={dateFilter.rangeType}
            onChange={(e) =>
              onDateFilterChange({
                ...dateFilter,
                rangeType: e.target.value as CatalogDateFilterState['rangeType'],
              })
            }
            className="bg-transparent text-xs text-slate-200 focus:outline-none cursor-pointer pr-2"
          >
            <option value="all" className="bg-obsidian-900 text-slate-100">
              All Dates
            </option>
            <option value="today" className="bg-obsidian-900 text-slate-100">
              Today
            </option>
            <option value="7d" className="bg-obsidian-900 text-slate-100">
              Last 7 Days
            </option>
            <option value="30d" className="bg-obsidian-900 text-slate-100">
              Last 30 Days
            </option>
            <option value="custom" className="bg-obsidian-900 text-slate-100">
              Custom Range
            </option>
          </select>
        </div>

        {dateFilter.rangeType === 'custom' && (
          <div className="flex items-center space-x-1.5 animate-fadeIn">
            <input
              type="date"
              value={dateFilter.startDate || ''}
              onChange={(e) => onDateFilterChange({ ...dateFilter, startDate: e.target.value })}
              className="bg-obsidian-900 border border-gold-500/20 text-xs text-slate-200 px-2 py-1 rounded-md focus:outline-none focus:border-gold-500/50"
            />
            <span className="text-slate-500 text-xs">-</span>
            <input
              type="date"
              value={dateFilter.endDate || ''}
              onChange={(e) => onDateFilterChange({ ...dateFilter, endDate: e.target.value })}
              className="bg-obsidian-900 border border-gold-500/20 text-xs text-slate-200 px-2 py-1 rounded-md focus:outline-none focus:border-gold-500/50"
            />
          </div>
        )}
      </div>
    </div>
  );
};
