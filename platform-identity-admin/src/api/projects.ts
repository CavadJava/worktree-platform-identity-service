import { platformIdentityApi } from './httpClients';
import type { Project } from './types';

export async function listProjects(): Promise<Project[]> {
  const response = await platformIdentityApi.get<Project[]>('/projects');
  return response.data;
}

export async function createProject(name: string): Promise<Project> {
  const response = await platformIdentityApi.post<Project>('/projects', { name });
  return response.data;
}
