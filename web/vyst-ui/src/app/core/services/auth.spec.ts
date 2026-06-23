import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { Apollo } from 'apollo-angular';
import { of } from 'rxjs';

import { AuthService } from './auth';

describe('AuthService', () => {
  let service: AuthService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideRouter([]),
        {
          provide: Apollo,
          useValue: {
            client: {
              resetStore: () => Promise.resolve(),
            },
            watchQuery: () => ({
              valueChanges: of({ data: { me: {} } }),
            }),
          },
        },
      ],
    });
    service = TestBed.inject(AuthService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
