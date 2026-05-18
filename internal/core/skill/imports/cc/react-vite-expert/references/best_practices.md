# React 最佳实践和模式

现代 React 开发模式、约定和要避免的反模式。

## 组件模式

### 1. 复合组件模式

允许组件一起工作，同时保持封装。

```typescript
// Tabs.tsx - Parent component
interface TabsContextValue {
  activeTab: string;
  setActiveTab: (tab: string) => void;
}

const TabsContext = createContext<TabsContextValue | null>(null);

export const Tabs = ({ defaultTab, children }: {
  defaultTab: string;
  children: React.ReactNode;
}) => {
  const [activeTab, setActiveTab] = useState(defaultTab);

  return (
    <TabsContext.Provider value={{ activeTab, setActiveTab }}>
      <div className="tabs">{children}</div>
    </TabsContext.Provider>
  );
};

// Sub-components
Tabs.List = ({ children }: { children: React.ReactNode }) => (
  <div className="tabs-list">{children}</div>
);

Tabs.Tab = ({ value, children }: { value: string; children: React.ReactNode }) => {
  const context = useContext(TabsContext);
  if (!context) throw new Error('Tab must be used within Tabs');

  return (
    <button
      className={context.activeTab === value ? 'active' : ''}
      onClick={() => context.setActiveTab(value)}
    >
      {children}
    </button>
  );
};

Tabs.Panel = ({ value, children }: { value: string; children: React.ReactNode }) => {
  const context = useContext(TabsContext);
  if (!context) throw new Error('Panel must be used within Tabs');

  return context.activeTab === value ? (
    <div className="tab-panel">{children}</div>
  ) : null;
};

// Usage
<Tabs defaultTab="profile">
  <Tabs.List>
    <Tabs.Tab value="profile">Profile</Tabs.Tab>
    <Tabs.Tab value="settings">Settings</Tabs.Tab>
  </Tabs.List>
  <Tabs.Panel value="profile">
    <ProfileContent />
  </Tabs.Panel>
  <Tabs.Panel value="settings">
    <SettingsContent />
  </Tabs.Panel>
</Tabs>
```

### 2. 渲染道具模式

使用值为函数的 prop 在组件之间共享代码。

```typescript
interface DataFetcherProps<T> {
  url: string;
  render: (data: { data: T | null; loading: boolean; error: Error | null }) => React.ReactNode;
}

function DataFetcher<T>({ url, render }: DataFetcherProps<T>) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    fetch(url)
      .then(res => res.json())
      .then(setData)
      .catch(setError)
      .finally(() => setLoading(false));
  }, [url]);

  return <>{render({ data, loading, error })}</>;
}

// Usage
<DataFetcher<User>
  url="/api/user"
  render={({ data, loading, error }) => {
    if (loading) return <Spinner />;
    if (error) return <Error message={error.message} />;
    return <UserProfile user={data} />;
  }}
/>
```

### 3. 高阶组件（HOC）

包装组件以添加功能。

```typescript
// withAuth.tsx - HOC
function withAuth<P extends object>(
  Component: React.ComponentType<P>
): React.FC<P> {
  return (props: P) => {
    const { user, loading } = useAuth();

    if (loading) return <Spinner />;
    if (!user) return <Navigate to="/login" />;

    return <Component {...props} />;
  };
}

// Usage
const ProtectedDashboard = withAuth(Dashboard);

// Can compose multiple HOCs
const EnhancedComponent = withAuth(withLogger(Dashboard));
```

### 4. 自定义 Hook 模式（优于 HOC）

提取并重用组件逻辑。

```typescript
// hooks/useAuth.ts
export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check auth status
    checkAuth().then(setUser).finally(() => setLoading(false));
  }, []);

  const login = async (credentials: Credentials) => {
    const user = await api.login(credentials);
    setUser(user);
  };

  const logout = async () => {
    await api.logout();
    setUser(null);
  };

  return { user, loading, login, logout };
}

// Usage in components
function Dashboard() {
  const { user, loading } = useAuth();

  if (loading) return <Spinner />;
  if (!user) return <Navigate to="/login" />;

  return <div>Welcome, {user.name}</div>;
}
```

### 5. 提供者模式

跨组件树共享数据，无需进行 prop 钻取。

```typescript
// contexts/ThemeContext.tsx
interface ThemeContextValue {
  theme: 'light' | 'dark';
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  const toggleTheme = () => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light');
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useTheme must be used within ThemeProvider');
  return context;
}

// Usage
<ThemeProvider>
  <App />
</ThemeProvider>

// In components
const { theme, toggleTheme } = useTheme();
```

## TypeScript 最佳实践

### 组件 Props 类型

```typescript
// ✅ Use interfaces for component props (extendable)
interface ButtonProps {
  variant?: 'primary' | 'secondary';
  size?: 'sm' | 'md' | 'lg';
  disabled?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
}

// ✅ Extend HTML attributes
interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

// ✅ Use discriminated unions for conditional props
type ButtonProps =
  | { variant: 'link'; href: string }
  | { variant: 'button'; onClick: () => void };

// ✅ Generic components
interface SelectProps<T> {
  options: T[];
  value: T;
  onChange: (value: T) => void;
  renderOption: (option: T) => React.ReactNode;
}

function Select<T>({ options, value, onChange, renderOption }: SelectProps<T>) {
  // Implementation
}
```

### 钩子打字

```typescript
// ✅ Type state explicitly when needed
const [user, setUser] = useState<User | null>(null);
const [count, setCount] = useState<number>(0);

// ✅ Type refs
const inputRef = useRef<HTMLInputElement>(null);
const timerRef = useRef<NodeJS.Timeout | null>(null);

// ✅ Type custom hooks
function useLocalStorage<T>(key: string, initialValue: T): [T, (value: T) => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch {
      return initialValue;
    }
  });

  const setValue = (value: T) => {
    setStoredValue(value);
    window.localStorage.setItem(key, JSON.stringify(value));
  };

  return [storedValue, setValue];
}

// ✅ Type event handlers
const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
  console.log(event.currentTarget);
};

const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  console.log(event.target.value);
};

const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
  event.preventDefault();
};
```

### 类型保护和缩小

```typescript
// Type guard function
function isUser(value: unknown): value is User {
  return (
    typeof value === 'object' &&
    value !== null &&
    'id' in value &&
    'name' in value
  );
}

// Usage
const data: unknown = await fetchData();
if (isUser(data)) {
  console.log(data.name); // TypeScript knows it's a User
}
```

## 错误处理模式

### 误差边界

```typescript
// ErrorBoundary.tsx
interface Props {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div>
          <h2>Something went wrong</h2>
          <details>
            <summary>Error details</summary>
            <pre>{this.state.error?.message}</pre>
          </details>
        </div>
      );
    }

    return this.props.children;
  }
}

// Usage
<ErrorBoundary
  fallback={<ErrorPage />}
  onError={(error) => logToService(error)}
>
  <App />
</ErrorBoundary>
```

### 异步错误处理

```typescript
// Hook for async operations with error handling
function useAsyncOperation<T>() {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = async (asyncFunc: () => Promise<T>) => {
    try {
      setLoading(true);
      setError(null);
      const result = await asyncFunc();
      setData(result);
      return result;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error');
      setError(error);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return { data, loading, error, execute };
}

// Usage
function Component() {
  const { data, loading, error, execute } = useAsyncOperation<User>();

  const handleFetch = () => {
    execute(() => fetchUser('123'))
      .then(user => console.log('Success:', user))
      .catch(error => console.error('Failed:', error));
  };

  if (loading) return <Spinner />;
  if (error) return <Error message={error.message} />;
  return <div>{data?.name}</div>;
}
```

## 表格处理

### 受控组件（推荐）

```typescript
function ContactForm() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    message: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    // Clear error when user starts typing
    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!formData.email.includes('@')) {
      newErrors.email = 'Invalid email';
    }
    if (formData.message.length < 10) {
      newErrors.message = 'Message must be at least 10 characters';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validate()) return;

    try {
      await api.submitForm(formData);
      // Reset form
      setFormData({ name: '', email: '', message: '' });
    } catch (error) {
      console.error('Submission failed:', error);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div>
        <input
          name="name"
          value={formData.name}
          onChange={handleChange}
          placeholder="Name"
        />
        {errors.name && <span className="error">{errors.name}</span>}
      </div>
      <div>
        <input
          name="email"
          value={formData.email}
          onChange={handleChange}
          placeholder="Email"
        />
        {errors.email && <span className="error">{errors.email}</span>}
      </div>
      <div>
        <textarea
          name="message"
          value={formData.message}
          onChange={handleChange}
          placeholder="Message"
        />
        {errors.message && <span className="error">{errors.message}</span>}
      </div>
      <button type="submit">Submit</button>
    </form>
  );
}
```

### 表单库（适用于复杂表单）

```typescript
// React Hook Form - Best performance
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const schema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email'),
  age: z.number().min(18, 'Must be 18+'),
});

type FormData = z.infer<typeof schema>;

function Form() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  const onSubmit = (data: FormData) => {
    console.log(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input {...register('name')} />
      {errors.name && <span>{errors.name.message}</span>}

      <input {...register('email')} />
      {errors.email && <span>{errors.email.message}</span>}

      <input type="number" {...register('age', { valueAsNumber: true })} />
      {errors.age && <span>{errors.age.message}</span>}

      <button type="submit">Submit</button>
    </form>
  );
}
```

## 测试最佳实践

### 使用 React 测试库进行组件测试

```typescript
// Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './Button';

describe('Button', () => {
  it('renders with text', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click me</Button>);

    fireEvent.click(screen.getByText('Click me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    render(<Button disabled>Click me</Button>);
    expect(screen.getByText('Click me')).toBeDisabled();
  });

  it('applies correct variant class', () => {
    const { container } = render(<Button variant="primary">Click me</Button>);
    expect(container.firstChild).toHaveClass('primary');
  });
});
```

### 定制挂钩测试

```typescript
// useCounter.test.ts
import { renderHook, act } from '@testing-library/react';
import { useCounter } from './useCounter';

describe('useCounter', () => {
  it('initializes with default value', () => {
    const { result } = renderHook(() => useCounter());
    expect(result.current.count).toBe(0);
  });

  it('initializes with custom value', () => {
    const { result } = renderHook(() => useCounter(10));
    expect(result.current.count).toBe(10);
  });

  it('increments count', () => {
    const { result } = renderHook(() => useCounter());

    act(() => {
      result.current.increment();
    });

    expect(result.current.count).toBe(1);
  });

  it('decrements count', () => {
    const { result } = renderHook(() => useCounter(5));

    act(() => {
      result.current.decrement();
    });

    expect(result.current.count).toBe(4);
  });
});
```

## 要避免的常见反模式

### ❌ 反模式 1：支柱钻井
```typescript
// Bad: Passing props through many levels
<App>
  <Header user={user} />
    <Nav user={user} />
      <UserMenu user={user} />
```

**✅ 解决方案：上下文或状态管理**
```typescript
<AuthProvider>
  <App>
    <Header />
      <Nav />
        <UserMenu /> {/* useAuth() hook */}
```

### ❌ 反模式 2：大型组件
```typescript
// Bad: 500+ line component doing everything
function Dashboard() {
  // Fetching
  // Filtering
  // Sorting
  // Rendering
  // Form handling
}
```

**✅解决方案：分成更小的组件**
```typescript
function Dashboard() {
  return (
    <DashboardLayout>
      <DashboardHeader />
      <DashboardStats />
      <DashboardChart />
      <DashboardTable />
    </DashboardLayout>
  );
}
```

### ❌ 反模式 3：直接改变状态
```typescript
// Bad: Direct mutation
const handleClick = () => {
  user.name = 'New Name'; // ❌
  setUser(user);
};
```

**✅ 解决方案：不可变更新**
```typescript
const handleClick = () => {
  setUser({ ...user, name: 'New Name' });
};
```

### ❌ 反模式 4：使用索引作为键
```typescript
// Bad: Using array index
{items.map((item, index) => (
  <div key={index}>{item.name}</div>
))}
```

**✅解决方案：使用唯一ID**
```typescript
{items.map((item) => (
  <div key={item.id}>{item.name}</div>
))}
```

### ❌反模式5：过度使用useEffect
```typescript
// Bad: Unnecessary useEffect
const [count, setCount] = useState(0);
const [doubled, setDoubled] = useState(0);

useEffect(() => {
  setDoubled(count * 2);
}, [count]);
```

**✅解决方案：导出状态**
```typescript
const [count, setCount] = useState(0);
const doubled = count * 2; // No useEffect needed
```

## 辅助功能 (a11y) 最佳实践

```typescript
// ✅ Semantic HTML
<button onClick={handleClick}>Click me</button>
// ❌ Not: <div onClick={handleClick}>Click me</div>

// ✅ ARIA labels
<button aria-label="Close modal" onClick={closeModal}>
  <CloseIcon />
</button>

// ✅ Keyboard navigation
<div
  role="button"
  tabIndex={0}
  onClick={handleClick}
  onKeyDown={(e) => e.key === 'Enter' && handleClick()}
>
  Custom Button
</div>

// ✅ Focus management
const buttonRef = useRef<HTMLButtonElement>(null);
useEffect(() => {
  buttonRef.current?.focus();
}, []);

<button ref={buttonRef}>Auto-focused</button>

// ✅ Alt text for images
<img src="avatar.jpg" alt="User avatar" />

// ✅ Form labels
<label htmlFor="email">Email</label>
<input id="email" type="email" />
```
