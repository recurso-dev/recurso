from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.create_payment_order_body import CreatePaymentOrderBody
from ...models.error import Error
from ...models.payment_order import PaymentOrder
from ...types import Response


def _get_kwargs(
    *,
    body: CreatePaymentOrderBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/payments/order",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | PaymentOrder | None:
    if response.status_code == 200:
        response_200 = PaymentOrder.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | PaymentOrder]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreatePaymentOrderBody,
) -> Response[Error | PaymentOrder]:
    """Create a gateway payment order for an invoice

     Creates a payment order on the configured payment gateway. Public and rate-limited.
    Note: the response uses the gateway port's Go field names (capitalized) verbatim.

    Args:
        body (CreatePaymentOrderBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PaymentOrder]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: CreatePaymentOrderBody,
) -> Error | PaymentOrder | None:
    """Create a gateway payment order for an invoice

     Creates a payment order on the configured payment gateway. Public and rate-limited.
    Note: the response uses the gateway port's Go field names (capitalized) verbatim.

    Args:
        body (CreatePaymentOrderBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PaymentOrder
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreatePaymentOrderBody,
) -> Response[Error | PaymentOrder]:
    """Create a gateway payment order for an invoice

     Creates a payment order on the configured payment gateway. Public and rate-limited.
    Note: the response uses the gateway port's Go field names (capitalized) verbatim.

    Args:
        body (CreatePaymentOrderBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PaymentOrder]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreatePaymentOrderBody,
) -> Error | PaymentOrder | None:
    """Create a gateway payment order for an invoice

     Creates a payment order on the configured payment gateway. Public and rate-limited.
    Note: the response uses the gateway port's Go field names (capitalized) verbatim.

    Args:
        body (CreatePaymentOrderBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PaymentOrder
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
