import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <!-- Stats Card 1 -->
      <div class="bg-white overflow-hidden shadow-sm rounded-2xl border border-gray-100 hover:shadow-lg transition-all duration-300 transform hover:-translate-y-1 group">
        <div class="p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0 bg-indigo-50 rounded-xl p-3 group-hover:bg-indigo-100 transition-colors">
              <!-- Icon -->
              <svg class="h-8 w-8 text-indigo-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Total Users</dt>
                <dd class="text-3xl font-bold text-gray-900 tracking-tight">{{ stats?.totalUsers || 0 }}</dd>
                <dd class="text-xs text-green-600 mt-1 flex items-center font-medium">
                   <span class="bg-green-100 text-green-800 py-0.5 px-2 rounded-full flex items-center">
                     <svg class="h-3 w-3 mr-1" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M12 7a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0V8.414l-4.293 4.293a1 1 0 01-1.414 0L8 10.414l-4.293 4.293a1 1 0 01-1.414-1.414l5-5a1 1 0 011.414 0L11 10.586 14.586 7H12z" clip-rule="evenodd"></path></svg>
                     12%
                   </span>
                   <span class="ml-2 text-gray-400">from last month</span>
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <!-- Stats Card 2 -->
      <div class="bg-white overflow-hidden shadow-sm rounded-2xl border border-gray-100 hover:shadow-lg transition-all duration-300 transform hover:-translate-y-1 group">
        <div class="p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0 bg-green-50 rounded-xl p-3 group-hover:bg-green-100 transition-colors">
              <!-- Icon -->
              <svg class="h-8 w-8 text-green-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Active Tenants</dt>
                <dd class="text-3xl font-bold text-gray-900 tracking-tight">{{ stats?.activeTenants || 0 }}</dd>
                <dd class="text-xs text-green-600 mt-1 flex items-center font-medium">
                   <span class="bg-green-100 text-green-800 py-0.5 px-2 rounded-full flex items-center">
                     <svg class="h-3 w-3 mr-1" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M12 7a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0V8.414l-4.293 4.293a1 1 0 01-1.414 0L8 10.414l-4.293 4.293a1 1 0 01-1.414-1.414l5-5a1 1 0 011.414 0L11 10.586 14.586 7H12z" clip-rule="evenodd"></path></svg>
                     4 new
                   </span>
                   <span class="ml-2 text-gray-400">this week</span>
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <!-- Stats Card 3 -->
      <div class="bg-white overflow-hidden shadow-sm rounded-2xl border border-gray-100 hover:shadow-lg transition-all duration-300 transform hover:-translate-y-1 group">
        <div class="p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0 bg-yellow-50 rounded-xl p-3 group-hover:bg-yellow-100 transition-colors">
              <!-- Icon -->
              <svg class="h-8 w-8 text-yellow-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <div class="ml-5 w-0 flex-1">
              <dl>
                <dt class="text-sm font-medium text-gray-500 truncate">Requests (24h)</dt>
                <dd class="text-3xl font-bold text-gray-900 tracking-tight">{{ stats?.requests24h || 0 }}</dd>
                <dd class="text-xs text-gray-500 mt-1 font-medium">
                   Avg 188/hour
                </dd>
              </dl>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-8">
      <h2 class="text-lg leading-6 font-bold text-gray-900 mb-4">Recent Activity</h2>
      <div class="bg-white shadow-sm rounded-2xl border border-gray-100 p-6">
        <div class="flex flex-col space-y-6">
           <!-- Mock Activity Item -->
           <div class="flex items-center justify-between group">
              <div class="flex items-center">
                 <div class="h-10 w-10 rounded-full bg-blue-50 flex items-center justify-center text-blue-600 group-hover:bg-blue-100 transition-colors">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path></svg>
                 </div>
                 <div class="ml-4">
                    <p class="text-sm font-semibold text-gray-900">New user registered</p>

                 </div>
              </div>
              <span class="text-xs font-medium text-gray-400 bg-gray-50 px-2 py-1 rounded-full">2 mins ago</span>
           </div>
           
           <div class="border-t border-gray-50"></div>

           <!-- Mock Activity Item -->
           <div class="flex items-center justify-between group">
              <div class="flex items-center">
                 <div class="h-10 w-10 rounded-full bg-green-50 flex items-center justify-center text-green-600 group-hover:bg-green-100 transition-colors">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                 </div>
                 <div class="ml-4">
                    <p class="text-sm font-semibold text-gray-900">Tenant verified</p>
                    <p class="text-xs text-gray-500">Demo Corp verified their domain</p>
                 </div>
              </div>
              <span class="text-xs font-medium text-gray-400 bg-gray-50 px-2 py-1 rounded-full">1 hour ago</span>
           </div>
        </div>
      </div>
    </div>
  `
})
export class DashboardComponent implements OnInit {
  stats: any = null;
  private http = inject(HttpClient);

  ngOnInit() {
    // Mock stats for now, or fetch from API if available
    this.stats = {
      totalUsers: 124,
      activeTenants: 12,
      requests24h: 4521
    };

    // Uncomment when API is ready
    // this.http.get('/api/v1/stats').subscribe(data => this.stats = data);
  }
}
