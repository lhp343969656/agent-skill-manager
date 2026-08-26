import {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {RefreshCw, Package, Trash2, Loader2, X, AlertTriangle, RefreshCcw, CheckCircle2, Search, Pencil, Clock, FileText, Settings} from 'lucide-react';
import * as Dialog from '@radix-ui/react-dialog';
import {useAppStore} from '../stores/appStore';
import {UninstallSkill, CheckUpdate, UpdateSkill} from '../../wailsjs/go/main/App';
import {RateLimitHint, formatTime} from '../components/RateLimitHint';

interface SkillItem {
    id: string;
    displayName: string;
    sourceUrl: string;
    repositoryOwner: string;
    repositoryName: string;
    repositoryPath: string;
    currentVersionId: string;
    displayVersion: string;
    note: string;
    installedAt: string;
}

// 每个 Skill 的更新检查状态
interface UpdateState {
    status: 'idle' | 'checking' | 'check-done' | 'updating' | 'error';
    hasUpdate: boolean;
    latestVersion: string;
    latestNotes: string;
    message: string;
}

export function InstalledPage() {
    const {skills, loading, error, isConfigured, refreshSkills, refreshGitHubRateLimit, gitHubAuthMethod, updateSkillNote} = useAppStore();
    const navigate = useNavigate();
    const safeSkills: SkillItem[] = (skills || []) as SkillItem[];
    const [uninstalling, setUninstalling] = useState<SkillItem | null>(null);
    const [busy, setBusy] = useState(false);
    const [banner, setBanner] = useState<{type: 'success' | 'error'; text: string} | null>(null);
    // 查看更新内容弹窗
    const [viewingNotes, setViewingNotes] = useState<{skill: SkillItem; notes: string} | null>(null);
    // 修改备注名
    const [noteEditing, setNoteEditing] = useState<SkillItem | null>(null);
    const [noteText, setNoteText] = useState('');
    const [noteBusy, setNoteBusy] = useState(false);
    const authenticated = gitHubAuthMethod !== '';

    // 更新状态映射
    const [updateStates, setUpdateStates] = useState<Record<string, UpdateState>>({});

    const isLocal = (skill: SkillItem) => skill.repositoryOwner === 'local';

    const getUpdateState = (skill: SkillItem): UpdateState =>
        updateStates[skill.id] || {status: 'idle', hasUpdate: false, latestVersion: '', latestNotes: '', message: ''};

    const setSkillState = (id: string, state: Partial<UpdateState>) => {
        setUpdateStates((prev) => ({
            ...prev,
            [id]: {...(prev[id] || {status: 'idle', hasUpdate: false, latestVersion: '', latestNotes: '', message: ''}), ...state},
        }));
    };

    const handleCheckUpdate = async (skill: SkillItem) => {
        setSkillState(skill.id, {status: 'checking', message: ''});
        try {
            const result = await CheckUpdate(skill.id);
            if (result.hasUpdate) {
                setSkillState(skill.id, {
                    status: 'check-done',
                    hasUpdate: true,
                    latestVersion: result.latestVersion,
                    latestNotes: result.updateNotes || '',
                    message: `发现新版本：${result.latestVersion || '未知'}`,
                });
            } else if (result.checkError) {
                setSkillState(skill.id, {status: 'error', message: result.checkError});
            } else {
                setSkillState(skill.id, {
                    status: 'check-done',
                    hasUpdate: false,
                    message: '已是最新版本',
                });
            }
            // 检查更新会消耗额度，完成后刷新剩余次数
            await refreshGitHubRateLimit();
        } catch (e: any) {
            setSkillState(skill.id, {status: 'error', message: e?.message || String(e)});
        }
    };

    const handleUpdate = async (skill: SkillItem) => {
        setSkillState(skill.id, {status: 'updating', message: '更新中...'});
        try {
            await UpdateSkill(skill.id);
            await refreshSkills();
            setSkillState(skill.id, {
                status: 'check-done',
                hasUpdate: false,
                message: '更新成功',
            });
            // 更新技能会消耗额度，完成后刷新剩余次数
            await refreshGitHubRateLimit();
        } catch (e: any) {
            setSkillState(skill.id, {status: 'error', message: e?.message || String(e)});
        }
    };

    const handleUninstall = async () => {
        if (!uninstalling) return;
        setBusy(true);
        try {
            const result = await UninstallSkill(uninstalling.id);
            await refreshSkills();
            // 关闭确认弹窗
            setUninstalling(null);
            // 用页面横幅提示结果
            if (result.failed.length > 0) {
                setBanner({type: 'error', text: `已卸载 ${uninstalling.displayName}，但部分移除未完成：${result.failed.join('；')}`});
            } else {
                setBanner({type: 'success', text: `已卸载 ${uninstalling.displayName}` +
                    (result.removed.length > 0 ? `，并从 ${result.removed.join('、')} 移除链接` : '')});
            }
        } catch (e: any) {
            setUninstalling(null);
            setBanner({type: 'error', text: e?.message || String(e)});
        } finally {
            setBusy(false);
        }
    };

    const handleSaveNote = async () => {
        if (!noteEditing) return;
        setNoteBusy(true);
        try {
            await updateSkillNote(noteEditing.id, noteText);
            setNoteEditing(null);
            setBanner({type: 'success', text: `已更新「${noteEditing.displayName}」的备注名`});
        } catch (e: any) {
            setBanner({type: 'error', text: e?.message || String(e)});
        } finally {
            setNoteBusy(false);
        }
    };

    return (
        <div className="p-6">
            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900">已安装的共享 Skill</h1>
                    <p className="mt-1 text-sm text-gray-500">所有已启用 Agent 都能使用这些 Skill</p>
                </div>
                {isConfigured && (
                    <button
                        onClick={refreshSkills}
                        className="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                    >
                        <RefreshCw className="h-4 w-4"/>
                        刷新
                    </button>
                )}
            </div>

            {/* 操作结果横幅 */}
            {banner && (
                <div className={`mb-4 flex items-center justify-between rounded-md p-3 text-sm ${
                    banner.type === 'success' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
                }`}>
                    <span>{banner.text}</span>
                    <button onClick={() => setBanner(null)} className="ml-3 text-gray-400 hover:text-gray-600">
                        <X className="h-4 w-4"/>
                    </button>
                </div>
            )}

            {!isConfigured ? (
                <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 py-16 text-center">
                    <Package className="mb-3 h-10 w-10 text-gray-400"/>
                    <h2 className="text-lg font-medium text-gray-700">尚未配置共享目录</h2>
                    <p className="mt-1 text-sm text-gray-500">
                        请先选择共享 Skill 安装目录
                    </p>
                    <button
                        onClick={() => navigate('/settings')}
                        className="mt-4 flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                        <Settings className="h-4 w-4"/>
                        去设置
                    </button>
                </div>
            ) : loading ? (
                <div className="text-sm text-gray-500">加载中...</div>
            ) : error ? (
                <div className="rounded-md bg-red-50 p-4 text-sm text-red-700">{error}</div>
            ) : safeSkills.length === 0 ? (
                <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 py-16 text-center">
                    <Package className="mb-3 h-10 w-10 text-gray-400"/>
                    <h2 className="text-lg font-medium text-gray-700">还没有安装任何 Skill</h2>
                    <p className="mt-1 text-sm text-gray-500">前往「安装」页面从 GitHub 或本地安装第一个 Skill</p>
                </div>
            ) : (
                <div className="space-y-3">
                    {safeSkills.map((skill) => {
                        const state = getUpdateState(skill);
                        return (
                            <div key={skill.id} className="rounded-lg border border-gray-200 bg-white p-4">
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm font-semibold text-gray-900">{skill.note || skill.displayName}</span>
                                            {skill.note && (
                                                <span className="text-xs text-gray-400">（{skill.displayName}）</span>
                                            )}
                                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                                                isLocal(skill)
                                                    ? 'bg-orange-50 text-orange-700'
                                                    : 'bg-indigo-50 text-indigo-700'
                                            }`}>
                                                {isLocal(skill) ? '本地' : 'GitHub'}
                                            </span>
                                            <span className="text-xs text-gray-500">
                                                版本：{isLocal(skill) ? '本地' : (skill.displayVersion || '未知')}
                                            </span>
                                        </div>
                                        <div className="mt-1 text-xs text-gray-500">
                                            {isLocal(skill)
                                                ? skill.sourceUrl
                                                : `${skill.repositoryOwner}/${skill.repositoryName}`}
                                        </div>
                                        {formatTime(skill.installedAt) && (
                                            <div className="mt-1 flex items-center gap-1 text-xs text-gray-400">
                                                <Clock className="h-3 w-3"/>
                                                安装于 {formatTime(skill.installedAt)}
                                            </div>
                                        )}

                                        {/* 更新信息行 */}
                                        {state.status === 'checking' && (
                                            <div className="mt-2 flex items-center gap-1 text-xs text-gray-500">
                                                <Loader2 className="h-3 w-3 animate-spin"/>
                                                正在检查更新...
                                            </div>
                                        )}
                                        {state.status === 'check-done' && state.hasUpdate && (
                                            <div className="mt-2 flex items-center gap-2 text-xs">
                                                <span className="text-amber-600">发现新版本</span>
                                                {state.latestNotes && (
                                                    <button
                                                        onClick={() => setViewingNotes({skill, notes: state.latestNotes})}
                                                        className="inline-flex items-center gap-1 rounded-md border border-amber-300 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 hover:bg-amber-100"
                                                    >
                                                        <FileText className="h-3 w-3"/>
                                                        更新内容
                                                    </button>
                                                )}
                                                {!isLocal(skill) && (
                                                    <button
                                                        onClick={() => handleUpdate(skill)}
                                                        className="inline-flex items-center gap-1 rounded-md bg-indigo-600 px-2 py-1 text-xs font-medium text-white hover:bg-indigo-700"
                                                    >
                                                        <RefreshCcw className="h-3 w-3"/>
                                                        更新
                                                    </button>
                                                )}
                                            </div>
                                        )}
                                        {state.status === 'check-done' && !state.hasUpdate && (
                                            <div className="mt-2 flex items-center gap-1 text-xs text-green-600">
                                                <CheckCircle2 className="h-3 w-3"/>
                                                {state.message}
                                            </div>
                                        )}
                                        {state.status === 'updating' && (
                                            <div className="mt-2 flex items-center gap-1 text-xs text-indigo-600">
                                                <Loader2 className="h-3 w-3 animate-spin"/>
                                                {state.message}
                                            </div>
                                        )}
                                        {state.status === 'error' && (
                                            <>
                                                <div className="mt-2 flex items-center gap-1 text-xs text-red-600">
                                                    <AlertTriangle className="h-3 w-3"/>
                                                    {state.message}
                                                </div>
                                                <RateLimitHint message={state.message} authenticated={authenticated}/>
                                            </>
                                        )}
                                    </div>

                                    {/* 操作按钮 */}
                                    <div className="flex items-center gap-2">
                                        {!isLocal(skill) && state.status !== 'updating' && (
                                            <button
                                                onClick={() => handleCheckUpdate(skill)}
                                                disabled={state.status === 'checking'}
                                                className="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50"
                                            >
                                                <Search className="h-3.5 w-3.5"/>
                                                {state.status === 'checking' ? '检查中...' : '检查更新'}
                                            </button>
                                        )}
                                        <button
                                            onClick={() => { setNoteEditing(skill); setNoteText(skill.note || ''); }}
                                            className="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
                                        >
                                            <Pencil className="h-3.5 w-3.5"/>
                                            备注
                                        </button>
                                        <button
                                            onClick={() => setUninstalling(skill)}
                                            className="inline-flex items-center gap-1 rounded-md border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                                        >
                                            <Trash2 className="h-3.5 w-3.5"/>
                                            卸载
                                        </button>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* 卸载确认对话框 */}
            <Dialog.Root open={!!uninstalling} onOpenChange={(open) => {
                if (!open) {
                    setUninstalling(null);
                }
            }}>
                <Dialog.Portal>
                    <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40"/>
                    <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl">
                        <div className="flex items-start justify-between">
                            <Dialog.Title className="flex items-center gap-2 text-lg font-bold text-gray-900">
                                <AlertTriangle className="h-5 w-5 text-amber-500"/>
                                确认卸载
                            </Dialog.Title>
                            <Dialog.Close asChild>
                                <button className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                                    <X className="h-4 w-4"/>
                                </button>
                            </Dialog.Close>
                        </div>

                        <p className="mt-3 text-sm text-gray-600">
                            确定要卸载 <span className="font-semibold text-gray-900">{uninstalling?.displayName}</span> 吗？
                            这将：从所有已启用 Agent 移除该 Skill 的链接，并删除共享目录中的 Skill 文件。
                        </p>

                        <div className="mt-5 flex justify-end gap-2">
                            <Dialog.Close asChild>
                                <button className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">
                                    取消
                                </button>
                            </Dialog.Close>
                            <button
                                onClick={handleUninstall}
                                disabled={busy}
                                className="flex items-center gap-2 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                            >
                                {busy ? <Loader2 className="h-4 w-4 animate-spin"/> : <Trash2 className="h-4 w-4"/>}
                                {busy ? '卸载中...' : '确认卸载'}
                            </button>
                        </div>
                    </Dialog.Content>
                </Dialog.Portal>
            </Dialog.Root>

            {/* 修改备注名对话框 */}
            <Dialog.Root open={!!noteEditing} onOpenChange={(open) => { if (!open) setNoteEditing(null); }}>
                <Dialog.Portal>
                    <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40"/>
                    <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl">
                        <div className="flex items-start justify-between">
                            <Dialog.Title className="flex items-center gap-2 text-lg font-bold text-gray-900">
                                <Pencil className="h-5 w-5 text-indigo-500"/>
                                修改备注名
                            </Dialog.Title>
                            <Dialog.Close asChild>
                                <button className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                                    <X className="h-4 w-4"/>
                                </button>
                            </Dialog.Close>
                        </div>

                        <p className="mt-3 text-sm text-gray-600">
                            为「<span className="font-semibold text-gray-900">{noteEditing?.displayName}</span>」设置一个容易识别的备注名，留空则使用原名。
                        </p>

                        <input
                            autoFocus
                            value={noteText}
                            onChange={(e) => setNoteText(e.target.value)}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleSaveNote(); }}
                            placeholder={`例如：${noteEditing?.displayName || ''}`}
                            className="mt-3 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />

                        <div className="mt-5 flex justify-end gap-2">
                            <Dialog.Close asChild>
                                <button className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">
                                    取消
                                </button>
                            </Dialog.Close>
                            <button
                                onClick={handleSaveNote}
                                disabled={noteBusy}
                                className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                            >
                                {noteBusy ? <Loader2 className="h-4 w-4 animate-spin"/> : <Pencil className="h-4 w-4"/>}
                                {noteBusy ? '保存中...' : '保存'}
                            </button>
                        </div>
                    </Dialog.Content>
                </Dialog.Portal>
            </Dialog.Root>

            {/* 查看更新内容对话框 */}
            <Dialog.Root open={!!viewingNotes} onOpenChange={(open) => { if (!open) setViewingNotes(null); }}>
                <Dialog.Portal>
                    <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40"/>
                    <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl">
                        <div className="flex items-start justify-between">
                            <Dialog.Title className="flex items-center gap-2 text-lg font-bold text-gray-900">
                                <FileText className="h-5 w-5 text-amber-500"/>
                                更新内容
                            </Dialog.Title>
                            <Dialog.Close asChild>
                                <button className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                                    <X className="h-4 w-4"/>
                                </button>
                            </Dialog.Close>
                        </div>

                        <p className="mt-1 text-sm text-gray-500">
                            {viewingNotes?.skill.displayName}
                            {viewingNotes && getUpdateState(viewingNotes.skill).latestVersion
                                ? ` · 新版本 ${getUpdateState(viewingNotes.skill).latestVersion}`
                                : ''}
                        </p>

                        <div className="mt-3 max-h-[60vh] overflow-y-auto whitespace-pre-wrap rounded-md bg-gray-50 p-4 text-sm leading-relaxed text-gray-700">
                            {viewingNotes?.notes || '（无更新说明）'}
                        </div>

                        <div className="mt-5 flex justify-end">
                            <Dialog.Close asChild>
                                <button className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">
                                    关闭
                                </button>
                            </Dialog.Close>
                        </div>
                    </Dialog.Content>
                </Dialog.Portal>
            </Dialog.Root>
        </div>
    );
}
