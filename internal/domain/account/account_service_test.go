package account

//
//import (
//	"context"
//	"testing"
//
//	"github.com/blackcloro/transaction-processor/internal"
//
//	"github.com/stretchr/testify/mock"
//	"github.com/stretchr/testify/suite"
//
//	"github.com/blackcloro/transaction-processor/internal/domain/transaction"
//)
//
//type mockRepository struct {
//	mock.Mock
//}
//
//func (m *mockRepository) GetByID(ctx context.Context, id int64) (*Account, error) {
//	args := m.Called(ctx, id)
//	return args.Get(0).(*Account), args.Error(1)
//}
//
//func (m *mockRepository) Update(ctx context.Context, account *Account) error {
//	args := m.Called(ctx, account)
//	return args.Error(0)
//}
//
//func (m *mockRepository) UpdateBalance(ctx context.Context, id int, amount float64) error {
//	args := m.Called(ctx, id, amount)
//	return args.Error(0)
//}
//
//func (m *mockRepository) WithTransaction(ctx context.Context, fn func(Repository) (float64, error)) (float64, error) {
//	args := m.Called(ctx, fn)
//	return args.Get(0).(float64), args.Error(1)
//}
//
//type AccountServiceTestSuite struct {
//	suite.Suite
//	mockRepo *mockRepository
//	service  *Service
//}
//
//func (s *AccountServiceTestSuite) SetupTest() {
//	s.mockRepo = new(mockRepository)
//	s.service = NewService(s.mockRepo)
//}
//
//func (s *AccountServiceTestSuite) TestCheckAndProcessTransaction() {
//	testCases := []struct {
//		name            string
//		initialBalance  float64
//		transaction     *transaction.Transaction
//		expectedBalance float64
//		expectedError   error
//	}{
//		{
//			name:           "Successful win transaction",
//			initialBalance: 1000,
//			transaction: &transaction.Transaction{
//				TransactionID: "win-1",
//				AccountID:     1,
//				SourceType:    transaction.SourceTypeGame,
//				State:         transaction.StateWin,
//				Amount:        100,
//			},
//			expectedBalance: 1100,
//			expectedError:   nil,
//		},
//		{
//			name:           "Successful loss transaction",
//			initialBalance: 1000,
//			transaction: &transaction.Transaction{
//				TransactionID: "loss-1",
//				AccountID:     1,
//				SourceType:    transaction.SourceTypeGame,
//				State:         transaction.StateLost,
//				Amount:        50,
//			},
//			expectedBalance: 950,
//			expectedError:   nil,
//		},
//		{
//			name:           "Insufficient funds",
//			initialBalance: 1000,
//			transaction: &transaction.Transaction{
//				TransactionID: "loss-2",
//				AccountID:     1,
//				SourceType:    transaction.SourceTypeGame,
//				State:         transaction.StateLost,
//				Amount:        2000,
//			},
//			expectedBalance: 1000,
//			expectedError:   internal.ErrInsufficientFunds,
//		},
//	}
//
//	for _, tc := range testCases {
//		s.Run(tc.name, func() {
//			ctx := context.Background()
//
//			s.mockRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(account.Repository) (float64, error)")).
//				Return(tc.expectedBalance, tc.expectedError).
//				Run(func(args mock.Arguments) {
//					fn := args.Get(1).(func(Repository) (float64, error))
//					s.mockRepo.On("GetByID", ctx, int64(1)).Return(&Account{Balance: tc.initialBalance}, nil)
//					if tc.expectedError == nil {
//						s.mockRepo.On("UpdateBalance", ctx, int64(1), tc.expectedBalance).Return(nil)
//					}
//					fn(s.mockRepo)
//				})
//
//			newBalance, err := s.service.CheckAndProcessTransaction(ctx, 1, tc.transaction)
//
//			if tc.expectedError != nil {
//				s.ErrorIs(err, tc.expectedError)
//			} else {
//				s.NoError(err)
//				s.Equal(tc.expectedBalance, newBalance)
//			}
//
//			s.mockRepo.AssertExpectations(s.T())
//		})
//	}
//}
//
//func TestAccountServiceSuite(t *testing.T) {
//	suite.Run(t, new(AccountServiceTestSuite))
//}
