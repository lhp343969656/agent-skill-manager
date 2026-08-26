import {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {Bot, Search, RefreshCw, Loader2, Pencil, Settings} from 'lucide-react';
import {useAppStore} from '../stores/appStore';

export function AgentsPage() {
    const {agents, isConfigured, detectAgents, updateAgentEnabled, updateAgentSkillsPath, pickAgentSkillsDir, loading} = useAppStore();
    const navigate = useNavigate();
    const safeAgents = agents || [];
    const [detecting, setDetecting] = useState(false);
    const [togglingId, setTogglingId] = useState<string | null>(null);
    const [pickingId, setPickingId] = useState<string | null>(null);

    const handleDetect = async () => {
        setDetecting(true);
        try {
            await detectAgents();
        } finally {
            setDetecting(false);
        }
    };

    const handleToggle = async (agent: {id: string; enabled: boolean}) => {
        setTogglingId(agent.id);
        try {
            await updateAgentEnabled(agent.id, !agent.enabled);
        } finally {
            setTogglingId(null);
        }
    };

    const handleChangeDir = async (agent: {id: string; displayName: string}) => {
        const dir = await pickAgentSkillsDir();
        if (!dir) return;
        setPickingId(agent.id);
        try {
            await updateAgentSkillsPath(agent.id, dir);
        } finally {
            setPickingId(null);
        }
    };

    const btnText = detecting ? '查找中…' : '查找 Agent';

    return (
        <div className="p-6">
            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">Agent 管理</h1>
                    <p className="mt-1 text-sm text-gray-500">启用 Agent 后，全部共享 Skill 自动可用</p>
                </div>
                {isConfigured && (
                    <button
                        onClick={handleDetect}
                        disabled={detecting || loading}
                        className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {detecting ? <Loader2 className="h-4 w-4 animate-spin"/> : <RefreshCw className="h-4 w-4"/>}
                        {btnText}
                    </button>
                )}
            </div>

            {!isConfigured ? (
                <div className="flex flex-col items-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 p-8 text-center">
                    <p className="text-sm text-gray-600">请先配置共享 Skill 目录</p>
                    <button
                        onClick={() => navigate('/settings')}
                        className="mt-4 flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                        <Settings className="h-4 w-4"/>
                        去设置
                    </button>
                </div>
            ) : safeAgents.length === 0 ? (
                <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 py-16 text-center">
                    <Search className="mb-3 h-10 w-10 text-gray-400"/>
                    <h2 className="text-lg font-medium text-gray-700">尚未检测到 Agent</h2>
                    <p className="mt-1 text-sm text-gray-500">
                        点击右上角「查找 Agent」，自动扫描本机已安装的 Agent（如 Codex、OpenCode）
                    </p>
                </div>
            ) : (
                <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
                    <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                        <tr>
                            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500">Agent</th>
                            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500">Skill 目录</th>
                            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500">启用</th>
                        </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                        {safeAgents.map((agent) => (
                            <tr key={agent.id} className="hover:bg-gray-50">
                                <td className="px-4 py-3">
                                    <div className="flex items-center gap-2">
                                        <Bot className="h-4 w-4 text-indigo-500"/>
                                        <span className="text-sm font-medium text-gray-900">{agent.displayName}</span>
                                    </div>
                                </td>
                                <td className="px-4 py-3">
                                    <div className="flex items-center gap-2">
                                        <span className="max-w-[300px] truncate text-sm text-gray-600" title={agent.skillsPath}>{agent.skillsPath}</span>
                                        <button
                                            type="button"
                                            onClick={() => handleChangeDir(agent)}
                                            disabled={pickingId === agent.id || loading}
                                            title="修改技能目录"
                                            className="inline-flex shrink-0 items-center gap-1 rounded px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-60"
                                        >
                                            {pickingId === agent.id ? <Loader2 className="h-3.5 w-3.5 animate-spin"/> : <Pencil className="h-3.5 w-3.5"/>}
                                            修改
                                        </button>
                                    </div>
                                </td>
                                <td className="px-4 py-3">
                                    <div className="flex items-center gap-2">
                                        <button
                                            type="button"
                                            role="switch"
                                            aria-checked={agent.enabled}
                                            aria-label={`${agent.displayName} 启用开关`}
                                            onClick={() => handleToggle(agent)}
                                            disabled={togglingId === agent.id || loading}
                                            className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60 ${
                                                agent.enabled ? 'bg-indigo-600' : 'bg-gray-200'
                                            }`}
                                        >
                                            <span
                                                className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
                                                    agent.enabled ? 'translate-x-6' : 'translate-x-1'
                                                } ${
                                                    togglingId === agent.id ? 'animate-spin' : ''
                                                }`}
                                            />
                                        </button>
                                        <span className={`text-xs font-medium ${
                                            agent.enabled ? 'text-green-700' : 'text-gray-500'
                                        }`}>
                                            {agent.enabled ? '已启用' : '未启用'}
                                        </span>
                                    </div>
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}
