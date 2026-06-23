
import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ErrorHandlerService } from '../../../core/services/error-handler.service';
import { animate, style, transition, trigger } from '@angular/animations';

@Component({
  selector: 'app-toast',
  standalone: true,
  imports: [CommonModule],
  animations: [
    trigger('toastAnimation', [
      transition(':enter', [
        style({ transform: 'translateY(-20px) scale(0.95)', opacity: 0 }),
        animate('300ms cubic-bezier(0.16, 1, 0.3, 1)', style({ transform: 'translateY(0) scale(1)', opacity: 1 }))
      ]),
      transition(':leave', [
        animate('200ms ease-in', style({ transform: 'translateY(-20px) scale(0.95)', opacity: 0 }))
      ])
    ])
  ],
  template: `
    @if (activeError()) {
      <div 
        class="fixed top-6 left-1/2 transform -translate-x-1/2 z-50 flex items-center gap-4 px-6 py-4 rounded-2xl shadow-2xl border backdrop-blur-xl transition-all duration-300 min-w-[320px] max-w-md pointer-events-auto"
        [ngClass]="{
          'bg-red-50/90 border-red-200 text-red-900': activeError()?.type === 'error',
          'bg-amber-50/90 border-amber-200 text-amber-900': activeError()?.type === 'warning',
          'bg-blue-50/90 border-blue-200 text-blue-900': activeError()?.type === 'info'
        }"
        [@toastAnimation]
      >
        <!-- Icon -->
        <div class="flex-shrink-0">
          @if (activeError()?.type === 'error') {
            <div class="p-2 bg-red-100 rounded-full">
                <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
            </div>
          } @else if (activeError()?.type === 'warning') {
            <div class="p-2 bg-amber-100 rounded-full">
                <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
            </div>
          } @else {
            <div class="p-2 bg-blue-100 rounded-full">
                <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
            </div>
          }
        </div>

        <!-- Content -->
        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold truncate">
            {{ activeError()?.message }}
          </p>
          @if (activeError()?.code && activeError()?.type === 'error') {
            <p class="text-xs opacity-75 font-mono mt-0.5">
              Code: {{ activeError()?.code }}
            </p>
          }
        </div>

        <!-- Close Button -->
        <button 
            (click)="errorService.clear()"
            class="flex-shrink-0 -mr-2 p-2 rounded-lg hover:bg-black/5 transition-colors focus:outline-none focus:ring-2"
            [ngClass]="{
                'focus:ring-red-500': activeError()?.type === 'error',
                'focus:ring-amber-500': activeError()?.type === 'warning',
                'focus:ring-blue-500': activeError()?.type === 'info'
            }"
        >
          <svg class="w-4 h-4 opacity-50 hover:opacity-100" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
          </svg>
        </button>
      </div>
    }
  `
})
export class ToastComponent {
  errorService = inject(ErrorHandlerService);
  activeError = this.errorService.activeError;
}
