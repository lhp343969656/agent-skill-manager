import {NavLink} from 'react-router-dom';
import {Package, Download, Bot, Settings, Boxes} from 'lucide-react';
import {useAppStore} from '../stores/appStore';

const navItems = [
    {to: '/', label: '已安装', icon: Package, end: true},
    {to: '/install', label: '安装', icon: Download, end: false},
    {to: '/agents', label: 'Agents', icon: Bot, end: false},
    {to: '/settings', label: '设置', icon: Settings, end: false},
];

export function Sidebar() {
    const appName = useAppStore((s) => s.appName);

    return (
        <aside className="flex h-full w-56 shrink-0 flex-col border-r border-gray-200 bg-white">
            <div className="flex items-center gap-2 border-b border-gray-200 px-4 py-4">
                <Boxes className="h-6 w-6 text-indigo-600"/>
                <div>
                    <div className="text-sm font-semibold text-gray-900">{appName || 'Agent Skill Manager'}</div>
                    <div className="text-xs text-gray-500">共享 Skill 管理器</div>
                </div>
            </div>

            <nav className="flex-1 space-y-1 px-2 py-3">
                {navItems.map((item) => (
                    <NavLink
                        key={item.to}
                        to={item.to}
                        end={item.end}
                        className={({isActive}) =>
                            `flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                                isActive
                                    ? 'bg-indigo-50 text-indigo-700'
                                    : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                            }`
                        }
                    >
                        <item.icon className="h-4 w-4"/>
                        {item.label}
                    </NavLink>
                ))}
            </nav>

            <div className="border-t border-gray-200 px-4 py-3 text-xs text-gray-400">
                v{useAppStore((s) => s.version)}
            </div>
        </aside>
    );
}
