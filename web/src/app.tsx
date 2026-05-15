import { Route, Router } from '@solidjs/router';
import { ServerProvider } from './context/server';
import { SessionProvider } from './context/session';
import { PlanProvider } from './context/plan';
import { NoteProvider } from './context/note';
import { NotificationProvider } from './context/notification';
import { ThemeProvider } from './context/theme';
import UpdateNotification from './components/update-notification';
import Home from './pages/home';
import Chat from './pages/session';
import PlanList from './pages/plan-list';
import PlanDetail from './pages/plan-detail';
import PlanTasksPage from './pages/plan-tasks';
import TaskExecution from './pages/task-execution';
import NotesPage from './pages/notes';
import NoteDetailPage from './pages/note-detail';
import SettingsLayout from './pages/settings/layout';
import GeneralSettings from './pages/settings/general';
import ModelsSettings from './pages/settings/models';
import AboutSettings from './pages/settings/about';

export default function App() {
  return (
    <Router root={AppWrapper}>
      <Route path="/" component={Home} />
      <Route path="/session/:id" component={Chat} />
      <Route path="/plan" component={PlanList} />
      <Route path="/plan/:id" component={PlanDetail} />
      <Route path="/plan/:id/tasks" component={PlanTasksPage} />
      <Route path="/task/:id" component={TaskExecution} />
      <Route path="/notes" component={NotesPage} />
      <Route path="/notes/:id" component={NoteDetailPage} />
      <Route path="/settings" component={SettingsLayout}>
        <Route path="/" component={GeneralSettings} />
        <Route path="/models" component={ModelsSettings} />
        <Route path="/about" component={AboutSettings} />
      </Route>
    </Router>
  );
}

function AppWrapper(props: { children?: any }) {
  return (
    <ServerProvider>
      <ThemeProvider>
        <SessionProvider>
          <PlanProvider>
            <NoteProvider>
              <NotificationProvider>
                <div class="flex h-screen bg-[color:var(--bg-base)] text-zinc-100 antialiased">
                  {props.children}
                </div>
                <UpdateNotification />
              </NotificationProvider>
            </NoteProvider>
          </PlanProvider>
        </SessionProvider>
      </ThemeProvider>
    </ServerProvider>
  );
}