import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { AuthService } from '../../../core/services/auth';

@Component({
  selector: 'app-admin-layout',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="min-h-screen bg-gray-50 flex font-sans">
      <!-- Sidebar -->
      <aside class="fixed inset-y-0 left-0 w-72 bg-gray-900 text-white transition-transform duration-300 transform md:translate-x-0 z-30 shadow-2xl"
           [class.-translate-x-full]="!sidebarOpen">
        <div class="flex items-center justify-center h-20 bg-gray-900/50 backdrop-blur-sm border-b border-gray-800">
          <div class="flex items-center space-x-3">
             <div class="h-10 w-10 rounded-xl bg-gradient-to-br from-primary-500 to-secondary-500 flex items-center justify-center text-white font-bold text-xl shadow-lg ring-2 ring-white/10">V</div>
             <span class="text-2xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-white to-gray-400">Vyst Admin</span>
          </div>
        </div>
        
        <nav class="mt-8 px-4 space-y-2">
          <a routerLink="/admin/dashboard" routerLinkActive="bg-primary-600 text-white shadow-lg shadow-primary-900/20" 
             class="group flex items-center px-4 py-3.5 text-sm font-medium rounded-xl text-gray-400 hover:bg-gray-800 hover:text-white transition-all duration-200">
            <svg class="mr-3 h-5 w-5 flex-shrink-0 transition-colors duration-200 group-hover:text-white" routerLinkActive="text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
               <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
            </svg>
            Dashboard
          </a>
          <a routerLink="/admin/users" routerLinkActive="bg-primary-600 text-white shadow-lg shadow-primary-900/20" 
             class="group flex items-center px-4 py-3.5 text-sm font-medium rounded-xl text-gray-400 hover:bg-gray-800 hover:text-white transition-all duration-200">
            <svg class="mr-3 h-5 w-5 flex-shrink-0 transition-colors duration-200 group-hover:text-white" routerLinkActive="text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
               <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            Users
          </a>
          <a routerLink="/admin/tenants" routerLinkActive="bg-primary-600 text-white shadow-lg shadow-primary-900/20" 
             class="group flex items-center px-4 py-3.5 text-sm font-medium rounded-xl text-gray-400 hover:bg-gray-800 hover:text-white transition-all duration-200">
            <svg class="mr-3 h-5 w-5 flex-shrink-0 transition-colors duration-200 group-hover:text-white" routerLinkActive="text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
               <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
            </svg>
            Tenants
          </a>
        </nav>
        
        <div class="absolute bottom-0 w-full p-6 bg-gray-900/50 backdrop-blur-sm border-t border-gray-800">
           <button (click)="logout()" class="w-full flex items-center justify-center px-4 py-2.5 border border-transparent rounded-xl shadow-sm text-sm font-medium text-white bg-red-600/10 hover:bg-red-600 hover:text-white border-red-600/20 focus:outline-none transition-all duration-200 group">
             <svg class="mr-2 h-4 w-4 text-red-500 group-hover:text-white transition-colors" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
             </svg>
             Logout
           </button>
        </div>
      </aside>

      <!-- Main Content -->
      <div class="flex-1 flex flex-col md:pl-72 transition-all duration-300 min-h-screen">
        <header class="bg-white/80 backdrop-blur-md shadow-sm border-b border-gray-200/50 sticky top-0 z-20">
          <div class="max-w-7xl mx-auto py-4 px-4 sm:px-6 lg:px-8 flex justify-between items-center">
            <div class="flex items-center">
                <button (click)="toggleSidebar()" class="md:hidden mr-4 p-2 rounded-lg text-gray-400 hover:text-gray-500 hover:bg-gray-100 focus:outline-none transition-colors">
                  <span class="sr-only">Open sidebar</span>
                  <svg class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
                <h1 class="text-2xl font-bold text-gray-900 tracking-tight">Dashboard</h1>
            </div>
            
            <div class="flex items-center space-x-4">
               <button class="p-2 rounded-full text-gray-400 hover:text-primary-600 hover:bg-primary-50 focus:outline-none transition-colors relative">
                  <span class="sr-only">Notifications</span>
                  <span class="absolute top-2 right-2 block h-2 w-2 rounded-full bg-red-500 ring-2 ring-white"></span>
                  <svg class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                     <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
               </button>
               <div class="h-9 w-9 rounded-full bg-gradient-to-br from-primary-100 to-primary-200 flex items-center justify-center text-primary-700 font-bold border border-primary-200 shadow-sm cursor-pointer hover:ring-2 hover:ring-primary-500 hover:ring-offset-2 transition-all">
                  A
               </div>
            </div>
          </div>
        </header>

        <main class="flex-1 overflow-y-auto p-6 lg:p-8">
          <div class="max-w-7xl mx-auto animate-fade-in">
            <router-outlet></router-outlet>
          </div>
        </main>
      </div>
    </div>
  `
})
export class LayoutComponent {
  sidebarOpen = false;
  private authService = inject(AuthService);

  toggleSidebar() {
    this.sidebarOpen = !this.sidebarOpen;
  }

  logout() {
    this.authService.logout();
  }
}
