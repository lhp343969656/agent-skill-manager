import {useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {Download, Link2, Loader2, CheckCircle2, AlertTriangle, Star, FolderInput, Settings} from 'lucide-react';
import * as Tabs from '@radix-ui/react-tabs';
import {useAppStore} from '../stores/appStore';
import {ScanRepository, ListRepositorySkills, InstallSkill, InstallLocalSkill, PickLocalDirectory, PickLocalSkillFile} from '../../wailsjs/go/main/App';
import {main} from '../../wailsjs/go/models';
import {RateLimitHint} from '../components/RateLimitHint';

export function InstallPage() {
    const {isConfigured, refreshSkills} = useAppStore();
    const navigate = useNavigate();

    return (
        <div className="p-6">
            <h1 className="text-2xl font-bold text-gray-900">安装 Skill</h1>
            <p className="mt-1 text-sm text-gray-500">从 GitHub 仓库或本地文件夹安装 Skill</p>

            {!isConfigured ? (
                <div className="mt-8 flex flex-col items-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 p-8 text-center">
                    <p className="text-sm text-gray-600">
                        请先配置共享 Skill 目录，再开始安装
                    </p>
                    <button
                        onClick={() => navigate('/settings')}
                        className="mt-4 flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                        <Settings className="h-4 w-4"/>
                        去设置
                    </button>
                </div>
            ) : (
                <Tabs.Root defaultValue="github" className="mt-6">
                    <Tabs.List className="flex gap-1 border-b border-gray-200">
                        <Tabs.Trigger
                            value="github"
                            className="flex items-center gap-2 rounded-t-md px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 data-[state=active]:border-b-2 data-[state=active]:border-indigo-600 data-[state=active]:text-indigo-700"
                        >
                            <Link2 className="h-4 w-4"/>
                            GitHub
                        </Tabs.Trigger>
                        <Tabs.Trigger
                            value="local"
                            className="flex items-center gap-2 rounded-t-md px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 data-[state=active]:border-b-2 data-[state=active]:border-indigo-600 data-[state=active]:text-indigo-700"
                        >
                            <FolderInput className="h-4 w-4"/>
                            本地安装
                        </Tabs.Trigger>
                    </Tabs.List>

                    <Tabs.Content value="github" className="mt-5 max-w-2xl">
                        <GithubInstall refreshSkills={refreshSkills}/>
                    </Tabs.Content>
                    <Tabs.Content value="local" className="mt-5 max-w-2xl">
                        <LocalInstall refreshSkills={refreshSkills}/>
                    </Tabs.Content>
                </Tabs.Root>
            )}
        </div>
    );
}

// ---------- GitHub 安装 ----------

function GithubInstall({refreshSkills}: { refreshSkills: () => Promise<void> }) {
    const {gitHubAuthMethod} = useAppStore();
    const authenticated = gitHubAuthMethod !== '';
    const [url, setUrl] = useState('');
    const [note, setNote] = useState('');
    const [scanning, setScanning] = useState(false);
    const [loadingSkills, setLoadingSkills] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [scanResult, setScanResult] = useState<main.ScanResult | null>(null);
    const [selectedRef, setSelectedRef] = useState('');
    const [skills, setSkills] = useState<main.RepoSkill[] | null>(null);
    const [installingPath, setInstallingPath] = useState('');
    // 已成功安装的技能相对路径集合，用于在列表行内显示“已安装”状态
    const [installed, setInstalled] = useState<Set<string>>(new Set());
    const [error, setError] = useState<string | null>(null);

    const handleScan = async () => {
        setError(null);
        setScanResult(null);
        setSkills(null);
        setInstalled(new Set());
        setScanning(true);
        try {
            const result = await ScanRepository(url.trim());
            setScanResult(result);
            setSelectedRef(result.defaultRef);
            // 扫描仓库信息成功后，自动加载仓库内的技能列表
            setLoadingSkills(true);
            try {
                const skillList = await ListRepositorySkills(url.trim());
                setSkills(skillList || []);
            } catch (e2: any) {
                setSkills([]);
                setError(e2?.message || String(e2));
            } finally {
                setLoadingSkills(false);
            }
        } catch (e: any) {
            setError(e?.message || String(e));
        } finally {
            setScanning(false);
        }
    };

    const handleInstall = async (relPath: string) => {
        setError(null);
        setInstalling(true);
        setInstallingPath(relPath);
        try {
            await InstallSkill({url: url.trim(), gitRef: selectedRef, note, skillPath: relPath});
            // 安装成功后标记该技能已安装，保留列表，不清空不重新扫描
            setInstalled((prev) => new Set(prev).add(relPath));
            await refreshSkills();
        } catch (e: any) {
            setError(e?.message || String(e));
        } finally {
            setInstalling(false);
            setInstallingPath('');
        }
    };

    return (
        <div>
            <label className="mb-2 block text-sm font-medium text-gray-700">
                GitHub 仓库地址
            </label>
            <div className="flex gap-2">
                <div className="relative flex-1">
                    <Link2 className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400"/>
                    <input
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="https://github.com/owner/repo 或仓库子目录"
                        className="w-full rounded-md border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                    />
                </div>
                <button
                    onClick={handleScan}
                    disabled={!url.trim() || scanning}
                    className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                >
                    {scanning ? <Loader2 className="h-4 w-4 animate-spin"/> : <Download className="h-4 w-4"/>}
                    {scanning ? '扫描中...' : '扫描'}
                </button>
            </div>

            {error && (
                <>
                    <div className="mt-4 flex items-start gap-2 rounded-md bg-red-50 p-3 text-sm text-red-700">
                        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0"/>
                        {error}
                    </div>
                    <RateLimitHint message={error} authenticated={authenticated}/>
                </>
            )}

            {scanResult && (
                <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white">
                    <div className="border-b border-gray-200 bg-gray-50 px-4 py-3">
                        <div className="flex items-center gap-2">
                            <Star className="h-4 w-4 text-yellow-500"/>
                            <span className="text-sm font-semibold text-gray-900">
                                {scanResult.owner}/{scanResult.repo}
                            </span>
                        </div>
                        {scanResult.stars > 0 && (
                            <p className="mt-1 flex items-center gap-1 text-xs text-gray-500">
                                <Star className="h-3 w-3 fill-yellow-400 text-yellow-500"/>
                                {scanResult.stars.toLocaleString()} 颗星
                            </p>
                        )}
                        {scanResult.description && (
                            <p className="mt-1 text-xs text-gray-500">{scanResult.description}</p>
                        )}
                        {scanResult.subPath && (
                            <p className="mt-1 text-xs text-gray-500">子目录：{scanResult.subPath}</p>
                        )}
                    </div>

                    <div className="px-4 py-3">
                        <label className="mb-1 block text-xs font-medium text-gray-600">
                            版本 / 分支（Tag 或 Commit）
                        </label>
                        <select
                            value={selectedRef}
                            onChange={(e) => setSelectedRef(e.target.value)}
                            className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none"
                        >
                            <option value={scanResult.defaultRef}>默认分支：{scanResult.defaultRef}</option>
                            {(scanResult.versions || []).filter(v => v.Ref !== scanResult.defaultRef).map((v) => (
                                <option key={v.Ref} value={v.Ref}>
                                    {v.Kind === 'release' ? '🏷️' : '🔖'} {v.Display}
                                </option>
                            ))}
                        </select>
                        {(scanResult.versions || []).length > 0 && (
                            <p className="mt-2 text-xs text-gray-400">
                                检测到 {(scanResult.versions || []).length} 个版本（Release / Tag / Commit）
                            </p>
                        )}
                    </div>

                    <div className="border-t border-gray-100 px-4 py-3">
                        <label className="mb-1 block text-xs font-medium text-gray-600">
                            备注名（可选）
                        </label>
                        <input
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            placeholder="给安装的技能起一个容易识别的名字，留空则用原名"
                            className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                        />
                    </div>

                    <div className="border-t border-gray-100 px-4 py-3">
                        <div className="flex items-center gap-2">
                            <label className="text-xs font-semibold text-gray-700">
                                仓库内的技能
                            </label>
                            {loadingSkills && (
                                <span className="flex items-center gap-1 text-xs text-gray-500">
                                    <Loader2 className="h-3 w-3 animate-spin"/>
                                    正在扫描技能...
                                </span>
                            )}
                        </div>

                        {!loadingSkills && skills && skills.length > 0 && (
                            <ul className="mt-2 divide-y divide-gray-100 rounded-md border border-gray-200">
                                {skills.map((sk) => (
                                    <li key={sk.relPath} className="flex items-center justify-between gap-3 px-3 py-2">
                                        <div className="min-w-0">
                                            <div className="truncate text-sm font-medium text-gray-900">{sk.name}</div>
                                            <div className="truncate text-xs text-gray-400">{sk.relPath}</div>
                                        </div>
                                        {installed.has(sk.relPath) ? (
                                            <span className="flex shrink-0 items-center gap-1 rounded-md bg-green-100 px-3 py-1.5 text-xs font-medium text-green-700">
                                                <CheckCircle2 className="h-3.5 w-3.5"/>
                                                已安装
                                            </span>
                                        ) : (
                                            <button
                                                onClick={() => handleInstall(sk.relPath)}
                                                disabled={installing}
                                                className="flex shrink-0 items-center gap-1 rounded-md bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                                            >
                                                {installing && installingPath === sk.relPath
                                                    ? <Loader2 className="h-3.5 w-3.5 animate-spin"/>
                                                    : <Download className="h-3.5 w-3.5"/>}
                                                {installing && installingPath === sk.relPath ? '安装中...' : '安装'}
                                            </button>
                                        )}
                                    </li>
                                ))}
                            </ul>
                        )}

                        {!loadingSkills && skills && skills.length === 0 && (
                            <p className="mt-2 rounded-md bg-amber-50 p-3 text-xs text-amber-700">
                                该仓库中未找到 SKILL.md 技能文件。它可能是导航清单类仓库（只含链接），请前往仓库查看具体技能来源。
                            </p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

// ---------- 本地安装 ----------

function LocalInstall({refreshSkills}: { refreshSkills: () => Promise<void> }) {
    const [localPath, setLocalPath] = useState('');
    const [installing, setInstalling] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [installResult, setInstallResult] = useState<main.LocalInstallResult | null>(null);

    const handleInstall = async () => {
        setError(null);
        setInstallResult(null);
        setInstalling(true);
        try {
            const result = await InstallLocalSkill(localPath.trim());
            setInstallResult(result);
            await refreshSkills();
        } catch (e: any) {
            setError(e?.message || String(e));
        } finally {
            setInstalling(false);
        }
    };

    const handlePickDir = async () => {
        setError(null);
        try {
            const dir = await PickLocalDirectory();
            if (dir) {
                setLocalPath(dir);
            }
        } catch (e: any) {
            setError(e?.message || String(e));
        }
    };

    const handlePickFile = async () => {
        setError(null);
        try {
            const file = await PickLocalSkillFile();
            if (file) {
                setLocalPath(file);
            }
        } catch (e: any) {
            setError(e?.message || String(e));
        }
    };

    return (
        <div>
            <p className="mb-3 text-sm text-gray-600">
                选择本地文件夹（会自动扫描其中的 SKILL.md），或直接选择单个 SKILL.md 文件。
                重新安装同名 Skill 会覆盖更新。
            </p>

            {/* 选择按钮 */}
            <div className="flex gap-2">
                <button
                    onClick={handlePickDir}
                    className="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                    <FolderInput className="h-4 w-4 text-indigo-500"/>
                    选择文件夹
                </button>
                <button
                    onClick={handlePickFile}
                    className="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                    <FolderInput className="h-4 w-4 text-indigo-500"/>
                    选择文件
                </button>
            </div>

            {/* 已选路径 / 手动输入 */}
            <div className="mt-3">
                <label className="mb-2 block text-sm font-medium text-gray-700">
                    本地路径
                </label>
                <div className="flex gap-2">
                    <div className="relative flex-1">
                        <FolderInput className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400"/>
                        <input
                            value={localPath}
                            onChange={(e) => setLocalPath(e.target.value)}
                            placeholder="选择文件夹 / 文件，或手动输入路径"
                            className="w-full rounded-md border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:outline-none"
                        />
                    </div>
                    <button
                        onClick={handleInstall}
                        disabled={!localPath.trim() || installing}
                        className="flex items-center gap-2 rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                    >
                        {installing ? <Loader2 className="h-4 w-4 animate-spin"/> : <Download className="h-4 w-4"/>}
                        {installing ? '安装中...' : '安装'}
                    </button>
                </div>
            </div>

            {error && (
                <div className="mt-4 flex items-start gap-2 rounded-md bg-red-50 p-3 text-sm text-red-700">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0"/>
                    {error}
                </div>
            )}

            {installResult && (
                <div className="mt-6 overflow-hidden rounded-lg border border-green-200 bg-green-50">
                    <div className="flex items-center gap-2 px-4 py-3">
                        <CheckCircle2 className="h-5 w-5 text-green-600"/>
                        <span className="text-sm font-semibold text-green-800">本地安装成功</span>
                    </div>
                    <div className="px-4 pb-3 text-sm text-green-700">
                        <p>Skill 名称：<span className="font-mono">{installResult.skillName}</span></p>
                        {(installResult.syncedAgents || []).length > 0 && (
                            <p className="mt-1">已同步到：{(installResult.syncedAgents || []).join(', ')}</p>
                        )}
                        {(installResult.conflicts || []).length > 0 && (
                            <p className="mt-1 text-amber-700">
                                ⚠️ 存在冲突（未覆盖用户内容）：{(installResult.conflicts || []).join(', ')}
                            </p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
