from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.initiate_checkout_payment_response_200 import InitiateCheckoutPaymentResponse200
from ...types import Response


def _get_kwargs(
    id: UUID,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/checkout/{id}/pay".format(
            id=quote(str(id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | InitiateCheckoutPaymentResponse200 | None:
    if response.status_code == 200:
        response_200 = InitiateCheckoutPaymentResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if response.status_code == 503:
        response_503 = Error.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | InitiateCheckoutPaymentResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | InitiateCheckoutPaymentResponse200]:
    """Initiate payment for an invoice

     Creates a payment order on the currency-routed gateway for an unpaid invoice. The response tells the
    frontend which flow to drive: `stripe` returns a PaymentIntent `client_secret` + `publishable_key`
    for the Payment Element; `razorpay` returns the order id + `razorpay_key_id` for the Checkout.js
    modal. Settlement is always verified server-side — see `/checkout/{id}/success` and
    `/checkout/{id}/razorpay/verify`.

    Args:
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | InitiateCheckoutPaymentResponse200]
    """

    kwargs = _get_kwargs(
        id=id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Error | InitiateCheckoutPaymentResponse200 | None:
    """Initiate payment for an invoice

     Creates a payment order on the currency-routed gateway for an unpaid invoice. The response tells the
    frontend which flow to drive: `stripe` returns a PaymentIntent `client_secret` + `publishable_key`
    for the Payment Element; `razorpay` returns the order id + `razorpay_key_id` for the Checkout.js
    modal. Settlement is always verified server-side — see `/checkout/{id}/success` and
    `/checkout/{id}/razorpay/verify`.

    Args:
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | InitiateCheckoutPaymentResponse200
    """

    return sync_detailed(
        id=id,
        client=client,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | InitiateCheckoutPaymentResponse200]:
    """Initiate payment for an invoice

     Creates a payment order on the currency-routed gateway for an unpaid invoice. The response tells the
    frontend which flow to drive: `stripe` returns a PaymentIntent `client_secret` + `publishable_key`
    for the Payment Element; `razorpay` returns the order id + `razorpay_key_id` for the Checkout.js
    modal. Settlement is always verified server-side — see `/checkout/{id}/success` and
    `/checkout/{id}/razorpay/verify`.

    Args:
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | InitiateCheckoutPaymentResponse200]
    """

    kwargs = _get_kwargs(
        id=id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Error | InitiateCheckoutPaymentResponse200 | None:
    """Initiate payment for an invoice

     Creates a payment order on the currency-routed gateway for an unpaid invoice. The response tells the
    frontend which flow to drive: `stripe` returns a PaymentIntent `client_secret` + `publishable_key`
    for the Payment Element; `razorpay` returns the order id + `razorpay_key_id` for the Checkout.js
    modal. Settlement is always verified server-side — see `/checkout/{id}/success` and
    `/checkout/{id}/razorpay/verify`.

    Args:
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | InitiateCheckoutPaymentResponse200
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
        )
    ).parsed
