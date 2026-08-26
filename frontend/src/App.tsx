import {useEffect} from 'react';
import {HashRouter, Routes, Route, Navigate} from 'react-router-dom';
import {Sidebar} from './components/Sidebar';
import {InstalledPage} from './pages/InstalledPage';
import {InstallPage} from './pages/InstallPage';
import {AgentsPage} from './pages/AgentsPage';
import {SettingsPage} from './pages/SettingsPage';
import {useAppStore} from './stores/appStore';

function App() {
    const init = useAppStore((s) => s.init);

    useEffect(() => {
        init();
    }, [init]);

    return (
        <HashRouter>
            <div className="flex h-screen bg-gray-50">
                <Sidebar/>
                <main className="flex-1 overflow-y-auto">
                    <Routes>
                        <Route path="/" element={<InstalledPage/>}/>
                        <Route path="/install" element={<InstallPage/>}/>
                        <Route path="/agents" element={<AgentsPage/>}/>
                        <Route path="/settings" element={<SettingsPage/>}/>
                        <Route path="*" element={<Navigate to="/" replace/>}/>
                    </Routes>
                </main>
            </div>
        </HashRouter>
    );
}

export default App
