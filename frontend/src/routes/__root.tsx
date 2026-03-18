import { createRootRoute, Link, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Layout, Space, Switch } from 'antd';
import { MessageProvider } from '../components/MessageProvider';
import { AdminModeProvider } from '../context/AdminModeProvider';
import { DeploymentStreamProvider } from '../context/DeploymentStreamContext';
import { useAdminMode } from '../context/useAdminMode';

const { Header, Content, Footer } = Layout;

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <MessageProvider>
      <AdminModeProvider>
        <DeploymentStreamProvider>
          <Layout style={{ minHeight: '100vh' }}>
            <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ color: 'white', fontSize: '20px', fontWeight: 'bold' }}>
                <Link to="/" style={{ color: 'white', textDecoration: 'none' }}>Flink Admin</Link>
              </div>
              <HeaderAdminModeToggle />
            </Header>
            <Content style={{ padding: '24px', maxWidth: '90%', margin: '0 10%', flex: '1 0 auto' }}>
              <Outlet />
            </Content>
            <Footer style={{ textAlign: 'center' }}>
              Flink Admin {new Date().getFullYear()}
            </Footer>
            <TanStackRouterDevtools />
          </Layout>
        </DeploymentStreamProvider>
      </AdminModeProvider>
    </MessageProvider>
  );
}

function HeaderAdminModeToggle() {
  const { isAdminMode, setIsAdminMode } = useAdminMode();

  return (
    <Space size="small">
      <span style={{ color: 'rgba(255, 255, 255, 0.85)' }}>Admin mode</span>
      <Switch checked={isAdminMode} onChange={setIsAdminMode} />
    </Space>
  );
}
